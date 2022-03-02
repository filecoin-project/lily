package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/filecoin-project/go-bitfield"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	builtininit "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/lens"
)

var ActorRegistry *vm.ActorRegistry

func init() {
	ActorRegistry = filcns.NewActorRegistry()
}

var log = logging.Logger("lily/lens")

// GetMessagesForTipset returns a list of messages sent as part of pts (parent) with receipts found in ts (child).
// No attempt at deduplication of messages is made. A list of blocks with their corresponding messages is also returned - it contains all messages
// in the block regardless if they were applied during the state change.
func GetExecutedAndBlockMessagesForTipset(ctx context.Context, cs *store.ChainStore, sm *stmgr.StateManager, current, executed *types.TipSet) (*lens.TipSetMessages, error) {
	ctx, span := otel.Tracer("").Start(ctx, "GetExecutedAndBlockMessagesForTipSet")
	defer span.End()
	if !types.CidArrsEqual(current.Parents().Cids(), executed.Cids()) {
		return nil, xerrors.Errorf("current tipset (%s) is not on the same chain as executed (%s)", current.Key(), executed.Key())
	}

	getActorCode, err := MakeGetActorCodeFunc(ctx, cs.ActorStore(ctx), current, executed)
	if err != nil {
		return nil, err
	}

	// Build a lookup of which blocks each message appears in
	messageBlocks := map[cid.Cid][]cid.Cid{}
	for blockIdx, bh := range executed.Blocks() {
		blscids, secpkcids, err := cs.ReadMsgMetaCids(bh.Messages)
		if err != nil {
			return nil, xerrors.Errorf("read messages for block: %w", err)
		}

		for _, c := range blscids {
			messageBlocks[c] = append(messageBlocks[c], executed.Cids()[blockIdx])
		}

		for _, c := range secpkcids {
			messageBlocks[c] = append(messageBlocks[c], executed.Cids()[blockIdx])
		}
	}
	span.AddEvent("read block message metadata")

	bmsgs, err := cs.BlockMsgsForTipset(executed)
	if err != nil {
		return nil, xerrors.Errorf("block messages for tipset: %w", err)
	}

	span.AddEvent("read block messages for tipset")

	pblocks := executed.Blocks()
	if len(bmsgs) != len(pblocks) {
		// logic error somewhere
		return nil, xerrors.Errorf("mismatching number of blocks returned from block messages, got %d wanted %d", len(bmsgs), len(pblocks))
	}

	count := 0
	for _, bm := range bmsgs {
		count += len(bm.BlsMessages) + len(bm.SecpkMessages)
	}

	// Start building a list of completed message with receipt
	emsgs := make([]*lens.ExecutedMessage, 0, count)

	// bmsgs is ordered by block
	var index uint64
	for blockIdx, bm := range bmsgs {
		for _, blsm := range bm.BlsMessages {
			msg := blsm.VMMessage()
			// if a message ran out of gas while executing this is expected.
			toCode, found := getActorCode(msg.To)
			if !found {
				log.Warnw("failed to find TO actor", "height", current.Height().String(), "message", msg.Cid().String(), "actor", msg.To.String())
			}
			// we must always be able to find the sender, else there is a logic error somewhere.
			fromCode, found := getActorCode(msg.From)
			if !found {
				return nil, xerrors.Errorf("failed to find from actor %s height %d message %s", msg.From, current.Height(), msg.Cid())
			}
			emsgs = append(emsgs, &lens.ExecutedMessage{
				Cid:           blsm.Cid(),
				Height:        executed.Height(),
				Message:       msg,
				BlockHeader:   pblocks[blockIdx],
				Blocks:        messageBlocks[blsm.Cid()],
				Index:         index,
				FromActorCode: fromCode,
				ToActorCode:   toCode,
			})
			index++
		}

		for _, secm := range bm.SecpkMessages {
			msg := secm.VMMessage()
			toCode, found := getActorCode(msg.To)
			// if a message ran out of gas while executing this is expected.
			if !found {
				log.Warnw("failed to find TO actor", "height", current.Height().String(), "message", msg.Cid().String(), "actor", msg.To.String())
			}
			// we must always be able to find the sender, else there is a logic error somewhere.
			fromCode, found := getActorCode(msg.From)
			if !found {
				return nil, xerrors.Errorf("failed to find from actor %s height %d message %s", msg.From, current.Height(), msg.Cid())
			}
			emsgs = append(emsgs, &lens.ExecutedMessage{
				Cid:           secm.Cid(),
				Height:        executed.Height(),
				Message:       secm.VMMessage(),
				BlockHeader:   pblocks[blockIdx],
				Blocks:        messageBlocks[secm.Cid()],
				Index:         index,
				FromActorCode: fromCode,
				ToActorCode:   toCode,
			})
			index++
		}

	}
	span.AddEvent("built executed message list")

	// Retrieve receipts using a block from the child tipset
	rs, err := adt.AsArray(cs.ActorStore(ctx), current.Blocks()[0].ParentMessageReceipts)
	if err != nil {
		return nil, xerrors.Errorf("amt load: %w", err)
	}

	if rs.Length() != uint64(len(emsgs)) {
		// logic error somewhere
		return nil, xerrors.Errorf("mismatching number of receipts: got %d wanted %d", rs.Length(), len(emsgs))
	}

	// Create a skeleton vm just for calling ShouldBurn
	vmi, err := vm.NewVM(ctx, &vm.VMOpts{
		StateBase:      current.ParentState(),
		Epoch:          current.Height(),
		Bstore:         cs.StateBlockstore(),
		NtwkVersion:    DefaultNetwork.Version,
		Actors:         filcns.NewActorRegistry(),
		Syscalls:       sm.Syscalls,
		CircSupplyCalc: sm.GetVMCirculatingSupply,
	})
	if err != nil {
		return nil, xerrors.Errorf("creating temporary vm: %w", err)
	}

	parentStateTree, err := state.LoadStateTree(cs.ActorStore(ctx), executed.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load state tree: %w", err)
	}
	span.AddEvent("loaded parent state tree")

	// Receipts are in same order as BlockMsgsForTipset
	for _, em := range emsgs {
		var r types.MessageReceipt
		if found, err := rs.Get(em.Index, &r); err != nil {
			return nil, err
		} else if !found {
			return nil, xerrors.Errorf("failed to find receipt %d", em.Index)
		}
		em.Receipt = &r

		burn, err := vmi.ShouldBurn(ctx, parentStateTree, em.Message, em.Receipt.ExitCode)
		if err != nil {
			return nil, xerrors.Errorf("deciding whether should burn failed: %w", err)
		}

		em.GasOutputs = vm.ComputeGasOutputs(em.Receipt.GasUsed, em.Message.GasLimit, em.BlockHeader.ParentBaseFee, em.Message.GasFeeCap, em.Message.GasPremium, burn)

	}
	span.AddEvent("computed executed message gas usage")

	blkMsgs := make([]*lens.BlockMessages, len(current.Blocks()))
	for idx, blk := range current.Blocks() {
		msgs, smsgs, err := cs.MessagesForBlock(blk)
		if err != nil {
			return nil, err
		}
		blkMsgs[idx] = &lens.BlockMessages{
			Block:        blk,
			BlsMessages:  msgs,
			SecpMessages: smsgs,
		}
	}

	span.AddEvent("read messages for current block")

	return &lens.TipSetMessages{
		Executed: emsgs,
		Block:    blkMsgs,
	}, nil
}

func ParseParams(params []byte, method abi.MethodNum, actCode cid.Cid) (string, string, error) {
	m, found := ActorRegistry.Methods[actCode][method]
	if !found {
		return "", "", fmt.Errorf("unknown method %d for actor %s", method, actCode)
	}

	// if the actor method doesn't expect params don't parse them
	// messages can contain unexpected params and remain valid, we need to ignore this case for parsing.
	if m.Params == reflect.TypeOf(new(abi.EmptyValue)) {
		return "", m.Name, nil
	}

	p := reflect.New(m.Params.Elem()).Interface().(cbg.CBORUnmarshaler)
	if err := p.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
		actorName := builtin.ActorNameByCode(actCode)
		return "", m.Name, fmt.Errorf("cbor decode into %s %s:(%s.%d) failed: %v", m.Name, actorName, actCode, method, err)
	}

	b, err := MarshalWithOverrides(p, map[reflect.Type]marshaller{
		reflect.TypeOf(bitfield.BitField{}): bitfieldCountMarshaller,
	})

	return string(b), m.Name, err
}

func MethodAndParamsForMessage(m *types.Message, destCode cid.Cid) (string, string, error) {
	// Method is optional, zero means a plain value transfer
	if m.Method == 0 {
		return "Send", "", nil
	}

	if !destCode.Defined() {
		return "Unknown", "", xerrors.Errorf("missing actor code")
	}

	params, method, err := ParseParams(m.Params, m.Method, destCode)
	if method == "Unknown" {
		return "", "", xerrors.Errorf("unknown method for actor type %s: %d", destCode.String(), int64(m.Method))
	}
	if err != nil {
		log.Warnf("failed to parse parameters of message %s: %v", m.Cid, err)
		// this can occur when the message is not valid cbor
		return method, "", err
	}
	if params == "" {
		return method, "", nil
	}

	return method, params, nil
}

func ActorNameAndFamilyFromCode(c cid.Cid) (name string, family string, err error) {
	if !c.Defined() {
		return "", "", xerrors.Errorf("cannot derive actor name from undefined CID")
	}
	name = builtin.ActorNameByCode(c)
	if name == "<unknown>" {
		return "", "", xerrors.Errorf("cannot derive actor name from unknown CID: %s (maybe we need up update deps?)", c.String())
	}
	tokens := strings.Split(name, "/")
	if len(tokens) != 3 {
		return "", "", xerrors.Errorf("cannot parse actor name: %s from tokens: %s", name, tokens)
	}
	// network = tokens[0]
	// version = tokens[1]
	family = tokens[2]
	return
}

func MakeGetActorCodeFunc(ctx context.Context, store adt.Store, next, current *types.TipSet) (func(a address.Address) (cid.Cid, bool), error) {
	ctx, span := otel.Tracer("").Start(ctx, "MakeGetActorCodeFunc")
	defer span.End()
	nextStateTree, err := state.LoadStateTree(store, next.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load state tree: %w", err)
	}

	// Build a lookup of actor codes that exist after all messages in the current epoch have been executed
	actorCodes := map[address.Address]cid.Cid{}
	if err := nextStateTree.ForEach(func(a address.Address, act *types.Actor) error {
		actorCodes[a] = act.Code
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("iterate actors: %w", err)
	}

	nextInitActor, err := nextStateTree.GetActor(builtininit.Address)
	if err != nil {
		return nil, xerrors.Errorf("getting init actor: %w", err)
	}

	nextInitActorState, err := builtininit.Load(store, nextInitActor)
	if err != nil {
		return nil, xerrors.Errorf("loading init actor state: %w", err)
	}

	return func(a address.Address) (cid.Cid, bool) {
		_, innerSpan := otel.Tracer("").Start(ctx, "GetActorCode")
		defer innerSpan.End()
		// Shortcut lookup before resolving
		c, ok := actorCodes[a]
		if ok {
			return c, true
		}

		ra, found, err := nextInitActorState.ResolveAddress(a)
		if err != nil || !found {
			log.Warnw("failed to resolve actor address", "address", a.String())
			return cid.Undef, false
		}

		c, ok = actorCodes[ra]
		if ok {
			return c, true
		}

		// Fall back to looking in current state tree. This actor may have been deleted.
		currentStateTree, err := state.LoadStateTree(store, current.ParentState())
		if err != nil {
			log.Warnf("failed to load state tree: %v", err)
			return cid.Undef, false
		}

		act, err := currentStateTree.GetActor(a)
		if err != nil {
			log.Warnw("failed to find actor in state tree", "address", a.String(), "error", err.Error())
			return cid.Undef, false
		}

		return act.Code, true
	}, nil
}

type marshaller func(interface{}) ([]byte, error)

func MarshalWithOverrides(v interface{}, overrides map[reflect.Type]marshaller) ([]byte, error) {
	pwt := paramWrapperType{
		obj:     v,
		replace: overrides,
	}
	return pwt.MarshalJSON()
}

// wrapper type for overloading json marshal methods
type paramWrapperType struct {
	obj     interface{}
	replace map[reflect.Type]marshaller
}

func (wt *paramWrapperType) MarshalJSON() ([]byte, error) {
	v := reflect.ValueOf(wt.obj)
	t := v.Type()

	// if this is the type we want to override marshalling for, do the thing.
	rf, ok := wt.replace[t]
	if ok {
		return rf(wt.obj)
	}

	// if the type has its own marshaller use that
	if t.Implements(reflect.TypeOf((*json.Marshaler)(nil)).Elem()) {
		return json.Marshal(wt.obj)
	}

	switch t.Kind() {
	case reflect.Ptr:
		// unwrap pointer
		v = v.Elem()
		t = t.Elem()
		fallthrough

	case reflect.Struct:
		// if its a struct, walk its fields and recurse.
		m := make(map[string]interface{})
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).IsExported() {
				m[t.Field(i).Name] = &paramWrapperType{
					obj:     v.Field(i).Interface(),
					replace: wt.replace,
				}
			}
		}
		return json.Marshal(m)

	case reflect.Slice:
		// if its a slice of go types, marshal them, otherwise walk its indexes and recurse
		var out []interface{}
		if v.Len() > 0 {
			switch v.Index(0).Kind() {
			case
				reflect.Bool,
				reflect.String,
				reflect.Map,
				reflect.Float32, reflect.Float64,
				reflect.Complex64, reflect.Complex128,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return json.Marshal(v.Interface())
			default:
			}
		}
		for i := 0; i < v.Len(); i++ {
			out = append(out, &paramWrapperType{
				obj:     v.Index(i).Interface(),
				replace: wt.replace,
			})
		}
		return json.Marshal(out)

	default:
		return json.Marshal(wt.obj)
	}
}

// marshal go-bitfield to json with count value included.
var bitfieldCountMarshaller = func(v interface{}) ([]byte, error) {
	rle := v.(bitfield.BitField)
	r, err := rle.RunIterator()
	if err != nil {
		return nil, err
	}
	count, err := rle.Count()
	if err != nil {
		return nil, err
	}

	var ret = struct {
		Count uint64
		RLE   []uint64
	}{}
	if r.HasNext() {
		first, err := r.NextRun()
		if err != nil {
			return nil, err
		}
		if first.Val {
			ret.RLE = append(ret.RLE, 0)
		}
		ret.RLE = append(ret.RLE, first.Len)

		for r.HasNext() {
			next, err := r.NextRun()
			if err != nil {
				return nil, err
			}

			ret.RLE = append(ret.RLE, next.Len)
		}
	} else {
		ret.RLE = []uint64{0}
	}
	ret.Count = count
	return json.Marshal(ret)
}

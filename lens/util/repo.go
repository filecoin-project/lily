package util

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/filecoin-project/go-bitfield"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/state"
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

func ParseParams(params []byte, method abi.MethodNum, actCode cid.Cid) (string, string, error) {
	m, found := ActorRegistry.Methods[actCode][method]
	if !found {
		return "", "", fmt.Errorf("unknown method %d for actor %s", method, actCode)
	}

	// if the actor method doesn't expect params don't parse them
	// messages can contain unexpected params and remain valid, we need to ignore this case for parsing.
	if m.Params == reflect.TypeOf(new(abi.EmptyValue)) {
		return "", m.Num, nil
	}

	p := reflect.New(m.Params.Elem()).Interface().(cbg.CBORUnmarshaler)
	if err := p.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
		actorName := builtin.ActorNameByCode(actCode)
		return "", m.Num, fmt.Errorf("parse message params cbor decode into %s %s:(%s.%d) return (hex): %s failed: %w", m.Num, actorName, actCode, method, hex.EncodeToString(params), err)
	}

	b, err := MarshalWithOverrides(p, map[reflect.Type]marshaller{
		reflect.TypeOf(bitfield.BitField{}): bitfieldCountMarshaller,
	})
	if err != nil {
		return "", "", fmt.Errorf("parse message params method: %d actor code: %s params: %s failed: %w", method, actCode, hex.EncodeToString(params), err)
	}

	return string(b), m.Num, err
}

func ParseReturn(ret []byte, method abi.MethodNum, actCode cid.Cid) (string, string, error) {
	m, found := ActorRegistry.Methods[actCode][method]
	if !found {
		return "", "", fmt.Errorf("unknown method %d for actor %s", method, actCode)
	}

	// if the actor method doesn't expect returns don't parse them
	if m.Ret == reflect.TypeOf(new(abi.EmptyValue)) {
		return "", m.Num, nil
	}

	p := reflect.New(m.Ret.Elem()).Interface().(cbg.CBORUnmarshaler)
	if err := p.UnmarshalCBOR(bytes.NewReader(ret)); err != nil {
		actorName := builtin.ActorNameByCode(actCode)
		return "", m.Num, fmt.Errorf("parse message return cbor decode into %s %s:(%s.%d) return (hex): %s failed: %w", m.Num, actorName, actCode, method, hex.EncodeToString(ret), err)
	}

	b, err := MarshalWithOverrides(p, map[reflect.Type]marshaller{
		reflect.TypeOf(bitfield.BitField{}): bitfieldCountMarshaller,
	})
	if err != nil {
		return "", "", fmt.Errorf("parse message return method: %d actor code: %s return (hex): %s failed: %w", method, actCode, hex.EncodeToString(ret), err)
	}

	return string(b), m.Num, err

}

func MethodAndParamsForMessage(m *types.Message, destCode cid.Cid) (string, string, error) {
	// Method is optional, zero means a plain value transfer
	if m.Method == 0 {
		return "Send", "", nil
	}

	if !destCode.Defined() {
		return "Unknown", "", fmt.Errorf("missing actor code")
	}

	params, method, err := ParseParams(m.Params, m.Method, destCode)
	if method == "Unknown" {
		return "", "", fmt.Errorf("unknown method for actor type %s: %d", destCode.String(), int64(m.Method))
	}
	if err != nil {
		log.Warnf("failed to parse parameters of message %s: %v", m.Cid().String(), err)
		// this can occur when the message is not valid cbor
		return method, "", err
	}
	if params == "" {
		return method, "", nil
	}

	return method, params, nil
}

type MessageParamsReturn struct {
	MethodName string
	Params     string
	Return     string
}

func walkExecutionTrace(et *types.ExecutionTrace, trace *[]*MessageTrace) {
	for _, sub := range et.Subcalls {
		*trace = append(*trace, &MessageTrace{
			Message:   sub.Msg,
			Receipt:   sub.MsgRct,
			Error:     sub.Error,
			Duration:  sub.Duration,
			GasCharge: sub.GasCharges,
		})
		walkExecutionTrace(&sub, trace) //nolint:scopelint,gosec
	}
}

type MessageTrace struct {
	Message   *types.Message
	Receipt   *types.MessageReceipt
	Error     string
	Duration  time.Duration
	GasCharge []*types.GasTrace
}

func GetChildMessagesOf(m *lens.MessageExecution) []*MessageTrace {
	var out []*MessageTrace
	walkExecutionTrace(&m.Ret.ExecutionTrace, &out)
	return out
}

func ActorNameAndFamilyFromCode(c cid.Cid) (name string, family string, err error) {
	if !c.Defined() {
		return "", "", fmt.Errorf("cannot derive actor name from undefined CID")
	}
	name = builtin.ActorNameByCode(c)
	if name == "<unknown>" {
		return "", "", fmt.Errorf("cannot derive actor name from unknown CID: %s (maybe we need up update deps?)", c.String())
	}
	tokens := strings.Split(name, "/")
	if len(tokens) != 3 {
		return "", "", fmt.Errorf("cannot parse actor name: %s from tokens: %s", name, tokens)
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
		return nil, fmt.Errorf("load state tree: %w", err)
	}

	// Build a lookup of actor codes that exist after all messages in the current epoch have been executed
	actorCodes := map[address.Address]cid.Cid{}
	if err := nextStateTree.ForEach(func(a address.Address, act *types.Actor) error {
		actorCodes[a] = act.Code
		return nil
	}); err != nil {
		return nil, fmt.Errorf("iterate actors: %w", err)
	}

	nextInitActor, err := nextStateTree.GetActor(builtininit.Address)
	if err != nil {
		return nil, fmt.Errorf("getting init actor: %w", err)
	}

	nextInitActorState, err := builtininit.Load(store, nextInitActor)
	if err != nil {
		return nil, fmt.Errorf("loading init actor state: %w", err)
	}

	return func(a address.Address) (cid.Cid, bool) {
		// TODO accept a context, don't take the function context.
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

func MarshalWithOverrides(v interface{}, overrides map[reflect.Type]marshaller) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = nil
			err = fmt.Errorf("failed to override message param json marshaller: %v", r)
		}
	}()
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

	if t.Kind() == reflect.Ptr {
		// unwrap pointer
		v = v.Elem()
		t = t.Elem()
	}

	// if v is the zero value use default marshaling
	if !v.IsValid() {
		return json.Marshal(wt.obj)
	}
	// if v is typed zero value use default marshaling
	if v.IsZero() {
		return json.Marshal(wt.obj)
	}

	switch t.Kind() {
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

	// this struct matches the param schema used in network v14
	// see https://github.com/filecoin-project/lily/pull/821/files#r821851219
	var ret = struct {
		Count uint64   `json:"elemcount"`
		RLE   []uint64 `json:"rle"`
		Type  string   `json:"_type"`
	}{
		Type: "bitfield",
	}
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

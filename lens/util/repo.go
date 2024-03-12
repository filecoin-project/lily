package util

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/tasks/actorstate/market"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	builtin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
)

var ActorRegistry *vm.ActorRegistry

func init() {
	ActorRegistry = consensus.NewActorRegistry()
}

var log = logging.Logger("lily/lens")

type CBORByteArray struct {
	Params string
}

func ParseVmMessageParams(params []byte, paramsCodec uint64, method abi.MethodNum, actCode cid.Cid) (string, string, error) {
	m, found := ActorRegistry.Methods[actCode][method]
	if !found {
		// if the method wasn't found it is likely the case the method was one of:
		// https://github.com/filecoin-project/builtin-actors/blob/f5311fe735df4d9baf5f82d4b3db10f3c51688c4/actors/docs/README.md?plain=1#L31
		// so we just marshal the raw value to json and bail with a warning
		paramj, err := json.Marshal(params)
		if err != nil {
			return "", "", err
		}
		return string(paramj), builtin.ActorNameByCode(actCode), nil
	}
	// If the codec is 0, the parameters/return value are "empty".
	// If the codec is 0x55, it's bytes.
	if paramsCodec == 0 || paramsCodec == 0x55 {
		paramj, err := json.Marshal(CBORByteArray{Params: market.SanitizeLabel(string(params))})
		if err != nil {
			return "", "", err
		}
		return string(paramj), m.Name, nil
	}
	return ParseParams(params, method, actCode)
}

func ParseVmMessageReturn(ret []byte, retCodec uint64, method abi.MethodNum, actCode cid.Cid) (string, string, error) {
	m, found := ActorRegistry.Methods[actCode][method]
	if !found {
		// if the method wasn't found it is likely the case the method was one of:
		// https://github.com/filecoin-project/builtin-actors/blob/f5311fe735df4d9baf5f82d4b3db10f3c51688c4/actors/docs/README.md?plain=1#L31
		// so we just marshal the raw value to json and bail with a warning
		retJ, err := json.Marshal(ret)
		if err != nil {
			return "", "", err
		}
		return string(retJ), builtin.ActorNameByCode(actCode), nil
	}
	// If the codec is 0, the parameters/return value are "empty".
	// If the codec is 0x55, it's bytes.
	if retCodec == 0 || retCodec == 0x55 {
		retj, err := json.Marshal(CBORByteArray{Params: market.SanitizeLabel(string(ret))})
		if err != nil {
			return "", "", err
		}
		return string(retj), m.Name, nil
	}
	return ParseReturn(ret, method, actCode)
}

func ParseParams(params []byte, method abi.MethodNum, actCode cid.Cid) (_ string, _ string, err error) {
	m, found := ActorRegistry.Methods[actCode][method]
	if !found {
		// if the method wasn't found it is likely the case the method was one of:
		// https://github.com/filecoin-project/builtin-actors/blob/f5311fe735df4d9baf5f82d4b3db10f3c51688c4/actors/docs/README.md?plain=1#L31
		// so we just marshal the raw value to json and bail with a warning
		paramj, err := json.Marshal(params)
		if err != nil {
			return "", "", err
		}
		return string(paramj), method.String(), nil
	}

	// if the actor method doesn't expect params don't parse them
	// messages can contain unexpected params and remain valid, we need to ignore this case for parsing.
	if m.Params == reflect.TypeOf(new(abi.EmptyValue)) ||
		len(params) == 0 {
		return "", m.Name, nil
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("method %s ActorName %s ParseParams recovered from panic: %+v", m.Name, builtin.ActorNameByCode(actCode), r)
		}
	}()

	// this statement can panic if the message params do not implement CBORUnmarshaler, so we recover above.
	// see https://github.com/filecoin-project/go-state-types/pull/119 for context
	p := reflect.New(m.Params.Elem()).Interface().(cbg.CBORUnmarshaler)
	if err := p.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
		actorName := builtin.ActorNameByCode(actCode)
		return "", m.Name, fmt.Errorf("parse message params cbor decode into %s %s:(%s.%d) return (hex): %s failed: %w", m.Name, actorName, actCode, method, hex.EncodeToString(params), err)
	}

	b, err := MarshalWithOverrides(p, map[reflect.Type]marshaller{
		reflect.TypeOf(bitfield.BitField{}): bitfieldCountMarshaller,
	})
	if err != nil {
		return "", "", fmt.Errorf("parse message params method: %d actor code: %s params: %s failed: %w", method, actCode, hex.EncodeToString(params), err)
	}

	return string(b), m.Name, err
}

func ParseReturn(ret []byte, method abi.MethodNum, actCode cid.Cid) (_ string, _ string, err error) {
	m, found := ActorRegistry.Methods[actCode][method]
	if !found {
		return "", "", fmt.Errorf("unknown method %d for actor %s", method, actCode)
	}

	// if the actor method doesn't expect returns don't parse them
	if m.Ret == reflect.TypeOf(new(abi.EmptyValue)) ||
		len(ret) == 0 {
		return "", m.Name, nil
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("ParseReturn recovered from panic: %+v", r)
		}
	}()

	// this statement can panic if the message params do not implement CBORUnmarshaler, so we recover above.
	// see https://github.com/filecoin-project/go-state-types/pull/119 for context
	p := reflect.New(m.Ret.Elem()).Interface().(cbg.CBORUnmarshaler)
	if err := p.UnmarshalCBOR(bytes.NewReader(ret)); err != nil {
		actorName := builtin.ActorNameByCode(actCode)
		return "", m.Name, fmt.Errorf("parse message return cbor decode into %s %s:(%s.%d) return (hex): %s failed: %w", m.Name, actorName, actCode, method, hex.EncodeToString(ret), err)
	}

	b, err := MarshalWithOverrides(p, map[reflect.Type]marshaller{
		reflect.TypeOf(bitfield.BitField{}): bitfieldCountMarshaller,
	})
	if err != nil {
		return "", "", fmt.Errorf("parse message return method: %d actor code: %s return (hex): %s failed: %w", method, actCode, hex.EncodeToString(ret), err)
	}

	return string(b), m.Name, err

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

type MessageTrace struct {
	Message   types.MessageTrace
	Receipt   types.ReturnTrace
	GasCharge []*types.GasTrace
	Index     uint64
}

func GetChildMessagesOf(m *lens.MessageExecution) []*MessageTrace {
	var out []*MessageTrace
	index := uint64(0)
	walkExecutionTrace(&m.Ret.ExecutionTrace, &out, &index)
	return out
}

func walkExecutionTrace(et *types.ExecutionTrace, trace *[]*MessageTrace, index *uint64) {
	for _, sub := range et.Subcalls {
		*trace = append(*trace, &MessageTrace{
			Message:   sub.Msg,
			Receipt:   sub.MsgRct,
			GasCharge: sub.GasCharges,
			Index:     *index,
		})
		*index++
		walkExecutionTrace(&sub, trace, index) //nolint:scopelint,gosec
	}
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

func MakeGetActorCodeFunc(ctx context.Context, store adt.Store, child, parent *types.TipSet) (func(ctx context.Context, a address.Address) (cid.Cid, bool), error) {
	_, span := otel.Tracer("").Start(ctx, "MakeGetActorCodeFunc")
	defer span.End()

	childStateTree, err := state.LoadStateTree(store, child.ParentState())
	if err != nil {
		return nil, fmt.Errorf("loading child state: %w", err)
	}

	parentStateTree, err := state.LoadStateTree(store, parent.ParentState())
	if err != nil {
		return nil, fmt.Errorf("loading parent state: %w", err)
	}

	return func(ctx context.Context, a address.Address) (cid.Cid, bool) {
		_, innerSpan := otel.Tracer("").Start(ctx, "GetActorCode")
		defer innerSpan.End()

		act, err := childStateTree.GetActor(a)
		if err == nil {
			return act.Code, true
		}

		// look in parent state, the address may have been deleted in the transition from parent -> child state.
		log.Infof("failed to find actor %s in child init actor state (err: %s), falling back to parent", a, err)
		act, err = parentStateTree.GetActor(a)
		if err == nil {
			return act.Code, true
		}

		log.Infof("failed to find actor %s in parent state: %s", a, err)
		return cid.Undef, false
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

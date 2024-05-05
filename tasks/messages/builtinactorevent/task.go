package builtinactorevent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fxamacker/cbor/v2"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actors/builtinactor"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/chain/types"
)

var log = logging.Logger("lily/tasks/builtinactorevent")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

var (
	fields  map[string][]types.ActorEventBlock
	convert map[string]string
)

type KVEvent struct {
	Key   string
	Value string
}

const (
	INT    = "int"
	STRING = "string"
	CID    = "cid"
	BIGINT = "bigint"
)

func init() {
	targetEvents := []string{
		"verifier-balance",
		"allocation",
		"allocation-removed",
		"claim",
		"claim-updated",
		"claim-removed",
		"deal-published",
		"deal-activated",
		"deal-terminated",
		"deal-completed",
		"sector-precommitted",
		"sector-activated",
		"sector-updated",
		"sector-terminated",
	}

	filterFields := []types.ActorEventBlock{}
	for _, filteredEvent := range targetEvents {
		fieldByte, err := ipld.Encode(basicnode.NewString(filteredEvent), dagcbor.Encode)
		if err == nil {
			filterFields = append(filterFields, types.ActorEventBlock{Codec: 0x51, Value: fieldByte})
		}
	}

	fields = map[string][]types.ActorEventBlock{"$type": filterFields}

	// ------------ convert ------------
	// https://fips.filecoin.io/FIPS/fip-0083.html
	convert = map[string]string{
		"$type":        STRING,
		"verifier":     INT,
		"client":       INT,
		"balance":      BIGINT,
		"id":           INT,
		"provider":     INT,
		"piece-cid":    CID,
		"piece-size":   INT,
		"term-min":     INT,
		"term-max":     INT,
		"expiration":   INT,
		"term-start":   INT,
		"sector":       INT,
		"unsealed-cid": CID,
	}
}
func CborValueDecode(key string, value []byte) interface{} {
	var (
		resultSTR    string
		resultINT    int
		resultBIGINT types.BigInt
		resultCID    cid.Cid
		err          error
	)

	switch convert[key] {
	case STRING:
		err = cbor.Unmarshal(value, &resultSTR)
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v", err, key)
			return nil
		}
		return resultSTR
	case INT:
		err = cbor.Unmarshal(value, &resultINT)
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v", err, key)
			return nil
		}
		return resultINT
	case BIGINT:
		err = cbor.Unmarshal(value, &resultBIGINT)
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v", err, key)
			return nil
		}
		return resultBIGINT
	case CID:
		nd, err := ipld.DecodeUsingPrototype(value, dagcbor.Decode, bindnode.Prototype((*cid.Cid)(nil), nil))
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v, value: %v", err, key, value)
			return nil
		}
		resultCID = *bindnode.Unwrap(nd).(*cid.Cid)
		return resultCID
	}

	return nil
}

func HandleEventEntries(event *types.ActorEvent) (string, map[string]interface{}, []*KVEvent) {
	var eventsSlice []*KVEvent
	var eventType string
	actorEvent := make(map[string]interface{})
	for _, e := range event.Entries {
		if e.Codec != 0x51 { // 81
			continue
		}

		var kvEvent KVEvent
		kvEvent.Key = e.Key

		v := CborValueDecode(e.Key, e.Value)
		switch convert[e.Key] {
		case STRING:
			kvEvent.Value = v.(string)
			if kvEvent.Key == "$type" {
				eventType = kvEvent.Value
			}
		case INT:
			kvEvent.Value = strconv.Itoa(v.(int))
		case BIGINT:
			kvEvent.Value = v.(types.BigInt).String()
		case CID:
			if v != nil {
				kvEvent.Value = v.(cid.Cid).String()
			} else {
				kvEvent.Value = ""
			}
		}
		if kvEvent.Key != "$type" {
			actorEvent[kvEvent.Key] = kvEvent.Value
		}
		eventsSlice = append(eventsSlice, &kvEvent)
	}

	return eventType, actorEvent, eventsSlice
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "fevm_trace"),
		)
	}
	defer span.End()
	errs := []error{}

	tsKey := executed.Key()
	filter := &types.ActorEventFilter{
		TipSetKey: &tsKey,
		Fields:    fields,
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	events, err := t.node.GetActorEventsRaw(ctx, filter)
	if err != nil {
		log.Errorf("GetActorEventsRaw[pTs: %v, pHeight: %v, cTs: %v, cHeight: %v] err: %v", executed.Key().String(), executed.Height(), current.Key().String(), current.Height(), err)
		errs = append(errs, err)
	}

	var (
		builtInActorResult = make(builtinactor.BuiltInActorEvents, 0)
	)

	for evtIdx, event := range events {
		eventType, actorEvent, eventsSlice := HandleEventEntries(event)

		obj := builtinactor.BuiltInActorEvent{
			Height:    int64(executed.Height()),
			Cid:       event.MsgCid.String(),
			Emitter:   event.Emitter.String(),
			EventType: eventType,
			EventIdx:  int64(evtIdx),
		}

		re, jsonErr := json.Marshal(eventsSlice)
		if jsonErr == nil {
			obj.EventEntries = string(re)
		}

		payload, jsonErr := json.Marshal(actorEvent)
		if jsonErr == nil {
			obj.EventPayload = string(payload)
		}
		if obj.EventType != "" {
			builtInActorResult = append(builtInActorResult, &obj)
		}
	}

	if len(errs) > 0 {
		report.ErrorsDetected = fmt.Errorf("%v", errs)
	}

	return builtInActorResult, report, nil
}

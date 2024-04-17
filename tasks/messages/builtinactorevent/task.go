package builtinactorevent

import (
	"context"
	"fmt"
	"strconv"

	b64 "encoding/base64"
	"encoding/json"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actors/builtinactor"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/fxamacker/cbor/v2"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var log = logging.Logger("lily/tasks/mineractordump")

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
	// ------------ fields ------------
	const (
		VerifierBalance    = "cHZlcmlmaWVyLWJhbGFuY2U="     // verifier-balance
		Allocation         = "amFsbG9jYXRpb24="             // allocation
		AllocationRemoved  = "cmFsbG9jYXRpb24tcmVtb3ZlZA==" // allocation-removed
		Claim              = "ZWNsYWlt"                     // claim
		ClaimUpdated       = "bWNsYWltLXVwZGF0ZWQ="         // claim-updated
		ClaimRemoved       = "bWNsYWltLXJlbW92ZWQ="         // claim-removed
		DealPublished      = "ZGVhbC1wdWJsaXNoZWQ="         // deal-published
		DealActivated      = "bmRlYWwtYWN0aXZhdGVk"         // deal-activated
		DealTerminated     = "b2RlYWwtdGVybWluYXRlZA=="     // deal-terminated
		DealCompleted      = "bmRlYWwtY29tcGxldGVk"         // deal-completed
		SectorPrecommitted = "c3NlY3Rvci1wcmVjb21taXR0ZWQ=" // sector-precommitted
		SectorActivated    = "cHNlY3Rvci1hY3RpdmF0ZWQ="     // sector-activated
		SectorUpdated      = "bnNlY3Rvci11cGRhdGVk"         // sector-updated
		SectorTerminated   = "cXNlY3Rvci10ZXJtaW5hdGVk"     // sector-terminated
	)

	verifierBalanceByte, _ := b64.StdEncoding.DecodeString(VerifierBalance)
	allocationByte, _ := b64.StdEncoding.DecodeString(Allocation)
	allocationRemovedByte, _ := b64.StdEncoding.DecodeString(AllocationRemoved)
	claimByte, _ := b64.StdEncoding.DecodeString(Claim)
	claimUpdatedByte, _ := b64.StdEncoding.DecodeString(ClaimUpdated)
	claimRemovedByte, _ := b64.StdEncoding.DecodeString(ClaimRemoved)
	dealPublishedByte, _ := b64.StdEncoding.DecodeString(DealPublished)
	dealActivatedByte, _ := b64.StdEncoding.DecodeString(DealActivated)
	dealTerminatedByte, _ := b64.StdEncoding.DecodeString(DealTerminated)
	dealCompletedByte, _ := b64.StdEncoding.DecodeString(DealCompleted)
	sectorPrecommittedByte, _ := b64.StdEncoding.DecodeString(SectorPrecommitted)
	sectorActivatedByte, _ := b64.StdEncoding.DecodeString(SectorActivated)
	sectorUpdatedByte, _ := b64.StdEncoding.DecodeString(SectorUpdated)
	sectorTerminatedByte, _ := b64.StdEncoding.DecodeString(SectorTerminated)

	fields = map[string][]types.ActorEventBlock{
		"$type": []types.ActorEventBlock{
			{81, verifierBalanceByte},    // verifier-balance
			{81, allocationByte},         // allocation
			{81, allocationRemovedByte},  // allocation-removed
			{81, claimByte},              // claim
			{81, claimUpdatedByte},       // claim-updated
			{81, claimRemovedByte},       // claim-removed
			{81, dealPublishedByte},      // deal-published
			{81, dealActivatedByte},      // deal-activated
			{81, dealTerminatedByte},     // deal-terminated
			{81, dealCompletedByte},      // deal-completed
			{81, sectorPrecommittedByte}, // sector-precommitted
			{81, sectorActivatedByte},    // sector-activated
			{81, sectorUpdatedByte},      // sector-updated
			{81, sectorTerminatedByte},   // sector-terminated
		},
	}

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
func cborValueDecode(key string, value []byte) interface{} {
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
		err = cbor.Unmarshal(value, &resultCID)
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v", err, key)
			return nil
		}
		return resultCID
	}

	return nil
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

	log.Errorf("Get the events count: %v", len(events))

	var (
		builtInActorResult = make(builtinactor.BuiltInActorEvents, 0)
	)

	for evtIdx, event := range events {
		var eventsSlice []*KVEvent
		var eventType string
		actorEvent := make(map[string]interface{})

		for entryIdx, e := range event.Entries {
			if e.Codec != 0x51 { // 81
				log.Warnf("Codec not equal to cbor, height: %v, evtIdx: %v, emitter: %v, entryIdx: %v, e.Codec: %v", executed.Height(), evtIdx, event.Emitter.String(), entryIdx, e.Codec)
				continue
			}

			var kvEvent KVEvent
			kvEvent.Key = e.Key

			v := cborValueDecode(e.Key, e.Value)
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
					kvEvent.Value = v.(string)
				}
			}
			if kvEvent.Key != "$type" {
				actorEvent[kvEvent.Key] = kvEvent.Value
			}
			eventsSlice = append(eventsSlice, &kvEvent)
		}

		obj := builtinactor.BuiltInActorEvent{
			Height:    int64(executed.Height()),
			Cid:       event.MsgCid.String(),
			Emitter:   event.Emitter.String(),
			EventType: eventType,
		}

		re, jsonErr := json.Marshal(eventsSlice)
		if jsonErr == nil {
			obj.EventEntries = string(re)
		}

		payload, jsonErr := json.Marshal(actorEvent)
		if jsonErr == nil {
			obj.EventPayload = string(payload)
		}

		builtInActorResult = append(builtInActorResult, &obj)
	}

	if len(errs) > 0 {
		report.ErrorsDetected = fmt.Errorf("%v", errs)
	}

	return builtInActorResult, report, nil
}

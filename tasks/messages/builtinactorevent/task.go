package builtinactorevent

import (
	"context"
	"encoding/json"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/lens/util"
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
	fields map[string][]types.ActorEventBlock
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

	fields = util.GenFilterFields(targetEvents)
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
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	_, err := t.node.MessageExecutions(ctx, current, executed)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting messages executions for tipset: %w", err)
		return nil, report, nil
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
		eventType, actorEvent, eventsSlice := util.HandleEventEntries(event)

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

package actorevent

import (
	"bytes"
	"context"
	"fmt"
	"math"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-amt-ipld/v4"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/messages"
)

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "actor_events"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	blkMsgRect, err := t.node.TipSetMessageReceipts(ctx, current, executed)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting tipset message receipet: %w", err)
		return nil, report, nil
	}

	var (
		out            = make(messagemodel.ActorEventList, 0, len(blkMsgRect))
		errorsDetected = make([]*messages.MessageError, 0, len(blkMsgRect))
		msgsSeen       = make(map[cid.Cid]bool, len(blkMsgRect))
	)

	for _, m := range blkMsgRect {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		itr, err := m.Iterator()
		if err != nil {
			return nil, nil, err
		}

		for itr.HasNext() {
			msg, _, rec := itr.Next()
			if msgsSeen[msg.Cid()] {
				continue
			}
			msgsSeen[msg.Cid()] = true

			if rec.EventsRoot == nil {
				continue
			}

			evtArr, err := amt.LoadAMT(ctx, t.node.Store(), *rec.EventsRoot, amt.UseTreeBitWidth(types.EventAMTBitwidth))
			if err != nil {
				report.ErrorsDetected = fmt.Errorf("loading actor events amt (%s): %w", *rec.EventsRoot, err)
				return nil, report, nil
			}
			var evt types.Event
			err = evtArr.ForEach(ctx, func(evtIdx uint64, deferred *cbg.Deferred) error {
				if evtIdx > math.MaxInt {
					return xerrors.Errorf("too many events")
				}
				if err := evt.UnmarshalCBOR(bytes.NewReader(deferred.Raw)); err != nil {
					return err
				}

				emitter, err := address.NewIDAddress(uint64(evt.Emitter))
				if err != nil {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid:   msg.Cid(),
						Error: fmt.Sprintf("failed to make ID address from event emitter (%s): %w", evt.Emitter, err),
					})
					return err
				}
				for _, e := range evt.Entries {
					out = append(out, &messagemodel.ActorEvent{
						Height:     int64(current.Height()),
						StateRoot:  current.ParentState().String(),
						MessageCid: msg.Cid().String(),
						EventIndex: int64(evtIdx),
						Emitter:    emitter.String(),
						Flags:      []byte{e.Flags},
						Key:        e.Key,
						Value:      e.Value,
					})
				}
				return nil
			})

			if err != nil {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   msg.Cid(),
					Error: fmt.Sprintf("loading actor events amt (%s): %w", *rec.EventsRoot, err),
				})
				continue
			}
		}
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		out,
	}, report, nil
}

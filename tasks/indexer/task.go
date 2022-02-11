package indexer

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/lens/task"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

func NewTask(node task.TaskAPI) *Task {
	return &Task{
		node: node,
	}
}

type Task struct {
	node task.TaskAPI
}

func (t *Task) ProcessTipSet(ctx context.Context, current *types.TipSet) (visormodel.ProcessingReportList, error) {
	executed, err := t.node.ChainGetTipSet(ctx, current.Parents())
	if err != nil {
		return nil, err
	}
	_, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("current", current.String()), attribute.Int64("height", int64(current.Height())))
		span.SetAttributes(attribute.String("executed", executed.String()), attribute.Int64("height", int64(executed.Height())))
	}
	defer span.End()

	rp := make(visormodel.ProcessingReportList, current.Height()-executed.Height())
	idx := 0
	for epoch := executed.Height(); epoch < current.Height(); epoch++ {
		if executed.Height() == epoch {
			rp[idx] = &visormodel.ProcessingReport{
				Height:    int64(epoch),
				StateRoot: executed.ParentState().String(),
			}
		} else {
			// null round no tipset
			rp[idx] = &visormodel.ProcessingReport{
				Height:            int64(epoch),
				StateRoot:         executed.ParentState().String(),
				StatusInformation: visormodel.ProcessingStatusInformationNullRound,
			}
		}
		idx += 1
	}
	return rp, nil
}

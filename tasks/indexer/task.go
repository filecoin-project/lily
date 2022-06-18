package indexer

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

type Task struct {
	node tasks.DataSource
}

func (t *Task) ProcessTipSet(ctx context.Context, current *types.TipSet) (visormodel.ProcessingReportList, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("processor", "indexer"),
		)
	}
	defer span.End()
	executed, err := t.node.TipSet(ctx, current.Parents())
	if err != nil {
		return nil, err
	}

	rp := make(visormodel.ProcessingReportList, current.Height()-executed.Height())
	idx := 0
	for epoch := current.Height(); epoch > executed.Height(); epoch-- {
		if current.Height() == epoch {
			rp[idx] = &visormodel.ProcessingReport{
				Height:    int64(epoch),
				StateRoot: current.ParentState().String(),
			}
		} else {
			// null round no tipset
			rp[idx] = &visormodel.ProcessingReport{
				Height:            int64(epoch),
				StateRoot:         executed.ParentState().String(),
				StatusInformation: visormodel.ProcessingStatusInformationNullRound,
			}
		}
		idx++
	}
	return rp, nil
}

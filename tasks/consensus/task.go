package consensus

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/chain"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/chain/types"
)

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (t *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	_, span := otel.Tracer("").Start(ctx, "ProcessTipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.Key().String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("processor", "consensus"),
		)
	}

	defer span.End()
	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	current := ts
	executed, err := t.node.TipSet(ctx, ts.Parents())
	if err != nil {
		return nil, nil, err
	}

	pl := make(chain.ChainConsensusList, current.Height()-executed.Height())
	idx := 0
	for epoch := current.Height(); epoch > executed.Height(); epoch-- {
		if current.Height() == epoch {
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: current.ParentState().String(),
				ParentTipSet:    current.Parents().String(),
				TipSet:          current.Key().String(),
			}
		} else {
			// null round no tipset
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: executed.ParentState().String(),
				ParentTipSet:    executed.Parents().String(),
				TipSet:          "",
			}
		}
		idx++
	}
	if executed.Height() == 0 {
		pl = append(pl, &chain.ChainConsensus{
			Height:          int64(executed.Height()),
			ParentStateRoot: executed.ParentState().String(),
			ParentTipSet:    executed.Parents().String(),
			TipSet:          executed.Key().String(),
		})
	}
	return pl, report, nil
}

package consensus

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/lens/task"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/chain"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

type Task struct {
	node task.TaskAPI
}

func NewTask(node task.TaskAPI) *Task {
	return &Task{
		node: node,
	}
}

func (t *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	child := ts
	parent, err := t.node.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return nil, nil, err
	}

	_, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("child", child.String()), attribute.Int64("height", int64(child.Height())))
		span.SetAttributes(attribute.String("parent", parent.String()), attribute.Int64("height", int64(parent.Height())))
	}
	defer span.End()

	pl := make(chain.ChainConsensusList, child.Height()-parent.Height())
	idx := 0
	for epoch := parent.Height(); epoch < child.Height(); epoch++ {
		if parent.Height() == epoch {
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: parent.ParentState().String(),
				ParentTipSet:    parent.Parents().String(),
				TipSet:          parent.Key().String(),
			}
		} else {
			// null round no tipset
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: parent.ParentState().String(),
				ParentTipSet:    parent.Parents().String(),
				TipSet:          "",
			}
		}
		idx += 1
	}
	return pl, report, nil
}

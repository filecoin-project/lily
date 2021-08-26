package blocks

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

type Task struct {
}

func NewTask() *Task {
	return &Task{}
}

func (p *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := global.Tracer("").Start(ctx, "ProcessBlocks")
	if span.IsRecording() {
		span.SetAttributes(label.String("tipset", ts.String()), label.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	var pl model.PersistableList
	for _, bh := range ts.Blocks() {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		pl = append(pl, blocks.NewBlockHeader(bh))
		pl = append(pl, blocks.NewBlockParents(bh))
		pl = append(pl, blocks.NewDrandBlockEntries(bh))
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	return pl, report, nil
}

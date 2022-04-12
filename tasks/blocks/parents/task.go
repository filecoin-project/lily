package parents

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

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
	_, span := otel.Tracer("").Start(ctx, "ProcessTipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.Key().String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("processor", "blocks"),
		)
	}

	var pl model.PersistableList
	for _, bh := range ts.Blocks() {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		pl = append(pl, blocks.NewBlockParents(bh))
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	return pl, report, nil
}

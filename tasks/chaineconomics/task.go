package chaineconomics

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/tasks")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	_, span := otel.Tracer("").Start(ctx, "ProcessTipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.Key().String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("processor", "chaineconomics"),
		)
	}
	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	ce, err := ExtractChainEconomicsModel(ctx, p.node, ts)
	if err != nil {
		log.Errorw("error received while extracting chain economics, closing lens", "error", err)
		return nil, nil, err
	}

	return ce, report, nil
}

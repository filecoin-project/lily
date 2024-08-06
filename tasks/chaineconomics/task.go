package chaineconomics

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	network2 "github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/chain/types"
)

var log = logging.Logger("lily/tasks")

type Task struct {
	node    tasks.DataSource
	version int
}

func NewTask(node tasks.DataSource, version int) *Task {
	return &Task{
		node:    node,
		version: version,
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

	if p.version == 2 {
		currentNetworkVersion := util.DefaultNetwork.Version(ctx, ts.Height())
		if currentNetworkVersion < network2.Version23 {
			return nil, nil, fmt.Errorf("The chain_economics_v2 will be supported in nv23. Current network version is %v", currentNetworkVersion)
		}
		ce, err := ExtractChainEconomicsV2Model(ctx, p.node, ts)
		if err != nil {
			log.Errorw("error received while extracting chain economics, closing lens", "error", err)
			return nil, nil, err
		}

		return ce, report, nil
	}

	ce, err := ExtractChainEconomicsModel(ctx, p.node, ts)
	if err != nil {
		log.Errorw("error received while extracting chain economics, closing lens", "error", err)
		return nil, nil, err
	}

	return ce, report, nil
}

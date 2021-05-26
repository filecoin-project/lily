package chaineconomics

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model"
	chainmodel "github.com/filecoin-project/sentinel-visor/model/chain"
	"github.com/filecoin-project/sentinel-visor/model/visor"
)

type EconomicsStorage interface {
	PersistBatch(ctx context.Context, ps ...model.Persistable) error
	MarkTipSetEconomicsComplete(ctx context.Context, tipset string, height int64, completedAt time.Time, errorsDetected string) error
	LeaseTipSetEconomics(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingTipSetList, error)
}

type ChainEconomicsLens interface {
	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (api.CirculatingSupply, error)
}

func ExtractChainEconomicsModel(ctx context.Context, node ChainEconomicsLens, ts *types.TipSet) (*chainmodel.ChainEconomics, error) {
	ctx, span := global.Tracer("").Start(ctx, "ExtractChainEconomics")
	if span.IsRecording() {
		span.SetAttributes(label.String("tipset", ts.String()), label.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	supply, err := node.StateVMCirculatingSupplyInternal(ctx, ts.Key())
	if err != nil {
		return nil, xerrors.Errorf("get circulating supply: %w", err)
	}

	return &chainmodel.ChainEconomics{
		ParentStateRoot: ts.ParentState().String(),
		VestedFil:       supply.FilVested.String(),
		MinedFil:        supply.FilMined.String(),
		BurntFil:        supply.FilBurnt.String(),
		LockedFil:       supply.FilLocked.String(),
		CirculatingFil:  supply.FilCirculating.String(),
	}, nil
}

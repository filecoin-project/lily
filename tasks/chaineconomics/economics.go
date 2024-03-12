package chaineconomics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	chainmodel "github.com/filecoin-project/lily/model/chain"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

type EconomicsStorage interface {
	PersistBatch(ctx context.Context, ps ...model.Persistable) error
	MarkTipSetEconomicsComplete(ctx context.Context, tipset string, height int64, completedAt time.Time, errorsDetected string) error
}

type ChainEconomicsLens interface {
	CirculatingSupply(context.Context, *types.TipSet) (api.CirculatingSupply, error)
}

func ExtractChainEconomicsModel(ctx context.Context, node ChainEconomicsLens, ts *types.TipSet) (*chainmodel.ChainEconomics, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ExtractChainEconomics")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.String()), attribute.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	supply, err := node.CirculatingSupply(ctx, ts)
	if err != nil {
		return nil, fmt.Errorf("get circulating supply: %w", err)
	}

	return &chainmodel.ChainEconomics{
		Height:              int64(ts.Height()),
		ParentStateRoot:     ts.ParentState().String(),
		VestedFil:           supply.FilVested.String(),
		MinedFil:            supply.FilMined.String(),
		BurntFil:            supply.FilBurnt.String(),
		LockedFil:           supply.FilLocked.String(),
		CirculatingFil:      supply.FilCirculating.String(),
		FilReserveDisbursed: supply.FilReserveDisbursed.String(),
	}, nil
}

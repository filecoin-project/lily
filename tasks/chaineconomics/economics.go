package chaineconomics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/go-address"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
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
	Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	Store() adt.Store
	MinerLoad(store adt.Store, act *types.Actor) (miner.State, error)
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

	chainEconomic := &chainmodel.ChainEconomics{
		Height:              int64(ts.Height()),
		ParentStateRoot:     ts.ParentState().String(),
		VestedFil:           supply.FilVested.String(),
		MinedFil:            supply.FilMined.String(),
		BurntFil:            supply.FilBurnt.String(),
		LockedFil:           supply.FilLocked.String(),
		CirculatingFil:      supply.FilCirculating.String(),
		FilReserveDisbursed: supply.FilReserveDisbursed.String(),
	}

	m, err := node.Actor(ctx, ts.MinTicketBlock().Miner, ts.Key())

	if err != nil {
		return chainEconomic, nil
	}

	minerState, err := node.MinerLoad(node.Store(), m)
	if err == nil && minerState.ActorVersion() >= actorstypes.Version14 {
		chainEconomic.LockedFilV2 = supply.FilLocked.String()
	}

	return chainEconomic, nil
}

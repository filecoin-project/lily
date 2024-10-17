package chaineconomics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	network2 "github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lily/lens/util"
	chainmodel "github.com/filecoin-project/lily/model/chain"

	"github.com/filecoin-project/lotus/chain/types"
)

func ExtractChainEconomicsV2Model(ctx context.Context, node ChainEconomicsLens, ts *types.TipSet) (*chainmodel.ChainEconomicsV2, error) {
	currentNetworkVersion := util.DefaultNetwork.Version(ctx, ts.Height())
	if currentNetworkVersion < network2.Version23 {
		log.Infof("The chain_economics_v2 will be supported in nv23. Current network version is %v", currentNetworkVersion)
		return nil, nil
	}

	ctx, span := otel.Tracer("").Start(ctx, "ExtractChainEconomicsV2")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.String()), attribute.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	supply, err := node.CirculatingSupply(ctx, ts)
	if err != nil {
		return nil, fmt.Errorf("get circulating supply: %w", err)
	}

	chainEconomicV2 := &chainmodel.ChainEconomicsV2{
		Height:              int64(ts.Height()),
		ParentStateRoot:     ts.ParentState().String(),
		VestedFil:           supply.FilVested.String(),
		MinedFil:            supply.FilMined.String(),
		BurntFil:            supply.FilBurnt.String(),
		LockedFilV2:         supply.FilLocked.String(),
		CirculatingFilV2:    supply.FilCirculating.String(),
		FilReserveDisbursed: supply.FilReserveDisbursed.String(),
	}

	return chainEconomicV2, nil
}

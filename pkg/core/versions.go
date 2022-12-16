package core

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
)

func ActorVersionForTipSet(ctx context.Context, ts *types.TipSet, ntwkVersionGetter func(ctx context.Context, epoch abi.ChainEpoch) network.Version) (actorstypes.Version, error) {
	ntwkVersion := ntwkVersionGetter(ctx, ts.Height())
	return actorstypes.VersionForNetwork(ntwkVersion)
}

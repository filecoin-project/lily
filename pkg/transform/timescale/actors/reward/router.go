package reward

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
	v0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v0"
	v2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v2"
	v3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v3"
	v4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v4"
	v5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v5"
	v6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v6"
	v7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v7"
	v8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v8"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward/v9"
)

func HandleReward(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *actordiff.ActorChange, version actortypes.Version) (model.Persistable, error) {
	switch version {
	case actortypes.Version0:
		return v0.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version2:
		return v2.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version3:
		return v3.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version4:
		return v4.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version5:
		return v5.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version6:
		return v6.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version7:
		return v7.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version8:
		return v8.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version9:
		return v9.RewardHandler(ctx, current, executed, addr, change)
	case actortypes.Version10:
		panic("not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported reward actor version: %d", version)
	}
}

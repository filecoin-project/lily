package minertransform

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lily/chain/actors"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
)

func State(ctx context.Context, stateDiff *minerdiff.StateDiff) (model.Persistable, error) {
	switch stateDiff.Miner.Actor.Code {
	case miner.VersionCodes()[actors.Version0]:
		V0MinerHandler(ctx, stateDiff)
	case miner.VersionCodes()[actors.Version2]:
	case miner.VersionCodes()[actors.Version3]:
	case miner.VersionCodes()[actors.Version4]:
	case miner.VersionCodes()[actors.Version5]:
	case miner.VersionCodes()[actors.Version6]:
	case miner.VersionCodes()[actors.Version7]:
	case miner.VersionCodes()[actors.Version8]:
	}
	return nil, fmt.Errorf("unsupported miner %s", stateDiff.Miner.Actor.Code)
}

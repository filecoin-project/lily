package raw

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	commonmodel "github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
)

func RawActorHandler(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *actordiff.ActorChange) (model.Persistable, error) {
	// noop when actor is removed
	if change.Change == core.ChangeTypeRemove {
		return nil, nil
	}

	state, err := json.Marshal(change.Current)
	if err != nil {
		return nil, err
	}

	return &commonmodel.ActorState{
		Height: int64(current.Height()),
		Head:   change.Actor.Head.String(),
		Code:   change.Actor.Code.String(),
		State:  string(state),
	}, nil
}

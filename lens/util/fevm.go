package util

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/tasks"
	builtin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
)

func IsEVMAddress(ctx context.Context, ds tasks.DataSource, addr address.Address, tsk types.TipSetKey) bool {
	act, err := ds.Actor(ctx, addr, tsk)
	if err != nil {
		// If actor not found, check if it's a placeholder address.
		if addr.Protocol() == address.Delegated {
			log.Debugf("Sent to Placeholder address: %v", addr)
			return true
		}
		log.Errorf("Error at getting actor. address: %v, err: %v", addr, err)
		return false
	}
	return builtin.IsEvmActor(act.Code)
}

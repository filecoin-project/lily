package v0

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
)

type Debt struct{}

func (Debt) Extract(ctx context.Context, current, parent *types.TipSet, addr address.Address, change *v0.StateDiffResult) (model.Persistable, error) {
	return &minermodel.MinerFeeDebt{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
		MinerID:   addr.String(),
		FeeDebt:   change.DebtChange.FeeDebt.String(),
	}, nil
}

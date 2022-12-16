package v9

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v9 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v9"
)

type Debt struct{}

func (Debt) Extract(ctx context.Context, current, parent *types.TipSet, addr address.Address, change *v9.StateDiffResult) (model.Persistable, error) {
	if change.DebtChange == nil {
		return nil, nil
	}
	return &minermodel.MinerFeeDebt{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
		MinerID:   addr.String(),
		FeeDebt:   change.DebtChange.FeeDebt.String(),
	}, nil
}

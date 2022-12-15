package v9

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
)

func HandleMinerDebtChange(ctx context.Context, store adt.Store, current, executed *types.TipSet, addr address.Address, changes *minerdiff.DebtChange) (model.Persistable, error) {
	return &minermodel.MinerFeeDebt{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
		MinerID:   addr.String(),
		FeeDebt:   changes.FeeDebt.String(),
	}, nil
}

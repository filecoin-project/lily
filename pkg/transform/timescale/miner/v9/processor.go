package v9

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
)

func ProcessMinerStateChanges(ctx context.Context, store adt.Store, current, executed *types.TipSet, addr address.Address, changes *minerdiff.StateDiff) (model.PersistableList, error) {
	var (
		infoModel  model.Persistable
		fundsModel model.Persistable
		debtModel  model.Persistable
		err        error
	)
	if changes.InfoChange != nil {
		infoModel, err = HandleMinerInfo(ctx, store, current, executed, addr, changes.InfoChange)
		if err != nil {
			return nil, err
		}
	}

	if changes.FundsChange != nil {
		fundsModel, err = HandleMinerFundsChange(ctx, store, current, executed, addr, changes.FundsChange)
		if err != nil {
			return nil, err
		}
	}

	if changes.DebtChange != nil {
		debtModel, err = HandleMinerDebtChange(ctx, store, current, executed, addr, changes.DebtChange)
		if err != nil {
			return nil, err
		}
	}

	sectorModel, err := HandleMinerSectorChanges(ctx, store, current, executed, addr, changes.SectorChanges)
	if err != nil {
		return nil, err
	}

	precommitModel, err := HandleMinerPreCommitChanges(ctx, store, current, executed, addr, changes.PreCommitChanges)
	if err != nil {
		return nil, err
	}

	sectorEventModel, err := HandleMinerSectorEvents(ctx, store, current, executed, addr, changes.PreCommitChanges, changes.SectorChanges, changes.SectorStatusChanges)
	if err != nil {
		return nil, err
	}

	sectorDealsModel, err := HandleMinerSectorDeals(ctx, store, current, executed, addr, changes.SectorChanges)
	if err != nil {
		return nil, err
	}
	return batchIfNotNil(infoModel, fundsModel, debtModel, sectorModel, precommitModel, sectorEventModel, sectorDealsModel), nil
}

func batchIfNotNil(models ...model.Persistable) model.PersistableList {
	out := make(model.PersistableList, 0, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

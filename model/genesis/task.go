package genesis

import (
	"context"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/go-pg/pg/v10"
)

type ProcessGenesisSingletonResult struct {
	minerResults []*GenesisMinerTaskResult
}

func (r *ProcessGenesisSingletonResult) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		for _, res := range r.minerResults {
			if err := res.StateModel.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := res.PowerModel.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := res.SectorModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := res.DealModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}

		}
		return nil
	})
}

func (r *ProcessGenesisSingletonResult) AddMiner(m *GenesisMinerTaskResult) {
	r.minerResults = append(r.minerResults, m)
}

type GenesisMinerTaskResult struct {
	StateModel   *miner.MinerState
	PowerModel   *miner.MinerPower
	SectorModels miner.MinerSectorInfos
	DealModels   miner.MinerDealSectors
}

package miner

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
)

type PreCommitInfoTransformer struct {
	meta v2.ModelMeta
}

func NewPrecommitInfoTransformer() *PreCommitInfoTransformer {
	info := miner.PreCommitEvent{}
	return &PreCommitInfoTransformer{meta: info.Meta()}
}

func (s *PreCommitInfoTransformer) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerPreCommitInfoList, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
				se := modeldata.(*miner.PreCommitEvent)
				// TODO add precommit removed event
				if se.Event != miner.PreCommitAdded {
					continue
				}
				sqlModels = append(sqlModels, &minermodel.MinerPreCommitInfo{
					Height:                 int64(se.Height),
					MinerID:                se.Miner.String(),
					SectorID:               uint64(se.Precommit.Info.SectorNumber),
					StateRoot:              se.StateRoot.String(),
					SealedCID:              se.Precommit.Info.SealedCID.String(),
					SealRandEpoch:          int64(se.Precommit.Info.SealRandEpoch),
					ExpirationEpoch:        int64(se.Precommit.Info.Expiration),
					PreCommitDeposit:       se.Precommit.PreCommitDeposit.String(),
					PreCommitEpoch:         int64(se.Precommit.PreCommitEpoch),
					DealWeight:             se.Precommit.DealWeight.String(),
					VerifiedDealWeight:     se.Precommit.VerifiedDealWeight.String(),
					IsReplaceCapacity:      se.Precommit.Info.ReplaceCapacity,
					ReplaceSectorDeadline:  se.Precommit.Info.ReplaceSectorDeadline,
					ReplaceSectorPartition: se.Precommit.Info.ReplaceSectorPartition,
					ReplaceSectorNumber:    uint64(se.Precommit.Info.ReplaceSectorNumber),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *PreCommitInfoTransformer) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *PreCommitInfoTransformer) Name() string {
	info := PreCommitInfoTransformer{}
	return reflect.TypeOf(info).Name()
}

func (s *PreCommitInfoTransformer) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

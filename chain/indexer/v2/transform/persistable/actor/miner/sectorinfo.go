package miner

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
)

type SectorInfoTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewSectorInfoTransform(taskName string) *SectorInfoTransform {
	info := miner.SectorEvent{}
	return &SectorInfoTransform{meta: info.Meta(), taskName: taskName}
}

func (s *SectorInfoTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debug("run SectorInfoTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerSectorInfoV7List, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				si := modeldata.(*miner.SectorEvent)
				if si.Event != miner.SectorAdded &&
					si.Event != miner.SectorExtended &&
					si.Event != miner.SectorSnapped {
					continue
				}
				sectorKeyCID := ""
				if si.SectorKeyCID != nil {
					sectorKeyCID = si.SectorKeyCID.String()
				}
				sqlModels = append(sqlModels, &minermodel.MinerSectorInfoV7{
					Height:                int64(si.Height),
					MinerID:               si.Miner.String(),
					SectorID:              uint64(si.SectorNumber),
					StateRoot:             si.StateRoot.String(),
					SealedCID:             si.SealedCID.String(),
					ActivationEpoch:       int64(si.Activation),
					ExpirationEpoch:       int64(si.Expiration),
					DealWeight:            si.DealWeight.String(),
					VerifiedDealWeight:    si.VerifiedDealWeight.String(),
					InitialPledge:         si.InitialPledge.String(),
					ExpectedDayReward:     si.ExpectedDayReward.String(),
					ExpectedStoragePledge: si.ExpectedStoragePledge.String(),
					SectorKeyCID:          sectorKeyCID,
				})
			}
			if len(sqlModels) > 0 {
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
		}
	}
	return nil
}

func (s *SectorInfoTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *SectorInfoTransform) Name() string {
	return reflect.TypeOf(SectorInfoTransform{}).Name()
}

func (s *SectorInfoTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

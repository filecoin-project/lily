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
	"github.com/filecoin-project/lily/tasks"
)

type SectorInfoTransform struct {
	meta v2.ModelMeta
}

func NewSectorInfoTransform() *SectorInfoTransform {
	info := miner.SectorEvent{}
	return &SectorInfoTransform{meta: info.Meta()}
}

func (s *SectorInfoTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debug("run SectorInfoTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerSectorInfoV7List, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
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
				out <- &persistable.Result{Model: sqlModels}
			}
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

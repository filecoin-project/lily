package miner

import (
	"context"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorevent"
	"github.com/filecoin-project/lily/tasks"
)

type SectorInfoTransform struct {
	Matcher v2.ModelMeta
}

func NewSectorInfoTransform() *SectorInfoTransform {
	info := sectorevent.SectorEvent{}
	return &SectorInfoTransform{Matcher: info.Meta()}
}

func (s SectorInfoTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerSectorInfoV7List, len(res.State().Data))
			for i, modeldata := range res.State().Data {
				si := modeldata.(*sectorevent.SectorEvent)
				if si.Event != sectorevent.SectorAdded &&
					si.Event != sectorevent.SectorExtended &&
					si.Event != sectorevent.SectorSnapped {
					continue
				}
				sectorKeyCID := ""
				if si.SectorKeyCID.Defined() {
					sectorKeyCID = si.SectorKeyCID.String()
				}
				sqlModels[i] = &minermodel.MinerSectorInfoV7{
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
				}
			}
			out <- &persistable.Result{Model: sqlModels}
		}
	}
	return nil
}

func (s SectorInfoTransform) ModelType() v2.ModelMeta {
	return s.Matcher
}

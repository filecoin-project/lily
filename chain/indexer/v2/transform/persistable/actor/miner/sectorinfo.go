package miner

import (
	"context"
	"sync"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorinfo"
	"github.com/filecoin-project/lily/tasks"
)

type PersistableResult struct {
	data model.Persistable
}

func (p *PersistableResult) Kind() transform.Kind {
	return "persistable"
}

func (p *PersistableResult) Data() interface{} {
	return p.data
}

func NewSectorInfoTransform() *SectorInfoTransform {
	info := sectorinfo.SectorInfo{}
	return &SectorInfoTransform{Matcher: info.Meta()}
}

type SectorInfoTransform struct {
	Matcher v2.ModelMeta
}

func (s SectorInfoTransform) Run(ctx context.Context, wg *sync.WaitGroup, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) {
	defer wg.Done()
	for res := range in {
		select {
		case <-ctx.Done():
			return
		default:
			sqlModels := make(minermodel.MinerSectorInfoV7List, len(res.State().Data))
			for i, modeldata := range res.State().Data {
				si := modeldata.(*sectorinfo.SectorInfo)
				sectorKeyCID := ""
				if !si.SealedCID.Equals(cid.Undef) {
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
			out <- &PersistableResult{data: sqlModels}
		}
	}
}

func (s SectorInfoTransform) ModelType() v2.ModelMeta {
	return s.Matcher
}

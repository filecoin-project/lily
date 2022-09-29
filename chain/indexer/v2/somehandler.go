package v2

import (
	"context"
	"sync"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorinfo"
	"github.com/filecoin-project/lily/tasks"
)

type PersistableResult struct {
	data model.Persistable
}

func (p *PersistableResult) Type() HandlerResultType {
	return "persistable"
}

func (p *PersistableResult) Data() interface{} {
	return p.data
}

func NewSectorInfoToPostgresHandler() *SectorInfoToPostgresHandler {
	info := sectorinfo.SectorInfo{}
	return &SectorInfoToPostgresHandler{Matcher: info.Meta()}
}

type SectorInfoToPostgresHandler struct {
	Matcher v2.ModelMeta
}

func (s SectorInfoToPostgresHandler) Run(ctx context.Context, wg *sync.WaitGroup, api tasks.DataSource, in chan *TipSetResult, out chan HandlerResult) {
	defer wg.Done()
	for res := range in {
		select {
		case <-ctx.Done():
			return
		default:
			sqlModels := make(minermodel.MinerSectorInfoV7List, len(res.Result.Data))
			for i, modeldata := range res.Result.Data {
				si := modeldata.(*sectorinfo.SectorInfo)
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
					SectorKeyCID:          si.SectorKeyCID.String(),
				}
			}
			out <- &PersistableResult{data: sqlModels}
			log.Infow("handler", "type", res.Task.String())
		}
	}
	log.Info("handler done")
}

func (s SectorInfoToPostgresHandler) ModelType() v2.ModelMeta {
	return s.Matcher
}

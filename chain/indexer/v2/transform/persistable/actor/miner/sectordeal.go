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

type SectorDealsTransformer struct {
	Matcher v2.ModelMeta
}

func NewSectorDealsTransformer() *SectorDealsTransformer {
	info := sectorevent.SectorEvent{}
	return &SectorDealsTransformer{Matcher: info.Meta()}
}

func (s *SectorDealsTransformer) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerSectorDealList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				se := modeldata.(*sectorevent.SectorEvent)
				if se.Event != sectorevent.SectorAdded && se.Event != sectorevent.SectorSnapped {
					continue
				}
				for _, dealIDs := range se.DealIDs {
					sqlModels = append(sqlModels, &minermodel.MinerSectorDeal{
						Height:   int64(se.Height),
						MinerID:  se.Miner.String(),
						SectorID: uint64(se.SectorNumber),
						DealID:   uint64(dealIDs),
					})

				}
			}
			out <- &persistable.Result{Model: sqlModels}
		}
	}
	return nil
}

func (s *SectorDealsTransformer) ModelType() v2.ModelMeta {
	return s.Matcher
}

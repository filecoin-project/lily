package miner

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorevent"
	"github.com/filecoin-project/lily/tasks"
)

type SectorDealsTransformer struct {
	meta v2.ModelMeta
}

func NewSectorDealsTransformer() *SectorDealsTransformer {
	info := sectorevent.SectorEvent{}
	return &SectorDealsTransformer{meta: info.Meta()}
}

func (s *SectorDealsTransformer) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debug("run SectorDealsTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("SectorDealsTransformer received data", "count", len(res.State().Data))
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
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *SectorDealsTransformer) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *SectorDealsTransformer) Name() string {
	info := SectorDealsTransformer{}
	return reflect.TypeOf(info).Name()
}

func (s *SectorDealsTransformer) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

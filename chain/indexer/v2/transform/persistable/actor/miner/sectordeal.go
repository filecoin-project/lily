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

type SectorDealsTransformer struct {
	meta     v2.ModelMeta
	taskName string
}

func NewSectorDealsTransformer(taskName string) *SectorDealsTransformer {
	info := miner.SectorEvent{}
	return &SectorDealsTransformer{meta: info.Meta(), taskName: taskName}
}

func (s *SectorDealsTransformer) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debug("run SectorDealsTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("SectorDealsTransformer received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerSectorDealList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				se := modeldata.(*miner.SectorEvent)
				if se.Event != miner.SectorAdded && se.Event != miner.SectorSnapped {
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
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
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

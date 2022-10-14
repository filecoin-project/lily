package miner

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
)

var log = logging.Logger("transform/miner")

type SectorEventTransformer struct {
	meta     v2.ModelMeta
	taskName string
}

func NewSectorEventTransformer(taskName string) *SectorEventTransformer {
	info := miner.SectorEvent{}
	return &SectorEventTransformer{meta: info.Meta(), taskName: taskName}
}

func (s *SectorEventTransformer) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("SectorEventTransformer received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerSectorEventList, len(res.Results.Models()))
			for i, modeldata := range res.Results.Models() {
				se := modeldata.(*miner.SectorEvent)
				sqlModels[i] = &minermodel.MinerSectorEvent{
					Height:    int64(se.Height),
					MinerID:   se.Miner.String(),
					SectorID:  uint64(se.SectorNumber),
					StateRoot: se.StateRoot.String(),
					Event:     se.Event.String(),
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

func (s *SectorEventTransformer) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *SectorEventTransformer) Name() string {
	info := SectorEventTransformer{}
	return reflect.TypeOf(info).Name()
}

func (s *SectorEventTransformer) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

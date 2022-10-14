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

type PrecommitEventTransformer struct {
	meta     v2.ModelMeta
	taskName string
}

func NewPrecommitEventTransformer(taskName string) *PrecommitEventTransformer {
	info := miner.PreCommitEvent{}
	return &PrecommitEventTransformer{meta: info.Meta(), taskName: taskName}
}

func (s *PrecommitEventTransformer) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debug("run PrecommitEventTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerSectorEventList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				se := modeldata.(*miner.PreCommitEvent)
				// TODO add precommit removed event
				if se.Event != miner.PreCommitAdded {
					continue
				}
				sqlModels = append(sqlModels, &minermodel.MinerSectorEvent{
					Height:    int64(se.Height),
					MinerID:   se.Miner.String(),
					SectorID:  uint64(se.Precommit.Info.SectorNumber),
					StateRoot: se.StateRoot.String(),
					Event:     se.Event.String(),
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

func (s *PrecommitEventTransformer) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *PrecommitEventTransformer) Name() string {
	info := PrecommitEventTransformer{}
	return reflect.TypeOf(info).Name()
}

func (s *PrecommitEventTransformer) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

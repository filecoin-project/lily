package miner

import (
	"context"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner/precommitevent"
	"github.com/filecoin-project/lily/tasks"
)

type PrecommitEventTransformer struct {
	Matcher v2.ModelMeta
}

func NewPrecommitEventTransformer() *PrecommitEventTransformer {
	info := precommitevent.PreCommitEvent{}
	return &PrecommitEventTransformer{Matcher: info.Meta()}
}

func (s *PrecommitEventTransformer) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Info("run PrecommitEventTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerSectorEventList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				se := modeldata.(*precommitevent.PreCommitEvent)
				// TODO add precommit removed event
				if se.Event != precommitevent.PreCommitAdded {
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
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *PrecommitEventTransformer) ModelType() v2.ModelMeta {
	return s.Matcher
}

func (s *PrecommitEventTransformer) Name() string {
	info := PrecommitEventTransformer{}
	return reflect.TypeOf(info).Name()
}

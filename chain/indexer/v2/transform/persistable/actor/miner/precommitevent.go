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
)

type PrecommitEventTransformer struct {
	meta v2.ModelMeta
}

func NewPrecommitEventTransformer() *PrecommitEventTransformer {
	info := miner.PreCommitEvent{}
	return &PrecommitEventTransformer{meta: info.Meta()}
}

func (s *PrecommitEventTransformer) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debug("run PrecommitEventTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerSectorEventList, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
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
				out <- &persistable.Result{Model: sqlModels}
			}
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

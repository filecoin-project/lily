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

type SectorEventTransformer struct {
	Matcher v2.ModelMeta
}

func NewSectorEventTransformer() *SectorEventTransformer {
	info := sectorevent.SectorEvent{}
	return &SectorEventTransformer{Matcher: info.Meta()}
}

func (s *SectorEventTransformer) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerSectorEventList, len(res.State().Data))
			for i, modeldata := range res.State().Data {
				se := modeldata.(*sectorevent.SectorEvent)
				sqlModels[i] = &minermodel.MinerSectorEvent{
					Height:    int64(se.Height),
					MinerID:   se.Miner.String(),
					SectorID:  uint64(se.SectorNumber),
					StateRoot: se.StateRoot.String(),
					Event:     se.Event.String(),
				}
			}
			out <- &persistable.Result{Model: sqlModels}
		}
	}
	return nil
}

func (s *SectorEventTransformer) ModelType() v2.ModelMeta {
	return s.Matcher
}

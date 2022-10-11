package miner

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("transform/sectorevents")

type SectorEventTransformer struct {
	meta v2.ModelMeta
}

func NewSectorEventTransformer() *SectorEventTransformer {
	info := miner.SectorEvent{}
	return &SectorEventTransformer{meta: info.Meta()}
}

func (s *SectorEventTransformer) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debug("run SectorEventTransformer")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("SectorEventTransformer received data", "count", len(res.State().Data))
			sqlModels := make(minermodel.MinerSectorEventList, len(res.State().Data))
			for i, modeldata := range res.State().Data {
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
				out <- &persistable.Result{Model: sqlModels}
			}
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

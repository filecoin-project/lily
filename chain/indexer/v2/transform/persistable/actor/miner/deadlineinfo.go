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

type DeadlineInfoTransform struct {
	meta v2.ModelMeta
}

func NewDeadlineInfoTransform() *DeadlineInfoTransform {
	info := miner.DeadlineInfo{}
	return &DeadlineInfoTransform{meta: info.Meta()}
}

func (s *DeadlineInfoTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerCurrentDeadlineInfoList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				di := modeldata.(*miner.DeadlineInfo)
				sqlModels = append(sqlModels, &minermodel.MinerCurrentDeadlineInfo{
					Height:        int64(di.Height),
					MinerID:       di.Miner.String(),
					StateRoot:     di.StateRoot.String(),
					DeadlineIndex: di.Index,
					PeriodStart:   int64(di.PeriodStart),
					Open:          int64(di.Open),
					Close:         int64(di.Close),
					Challenge:     int64(di.Challenge),
					FaultCutoff:   int64(di.FaultCutoff),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *DeadlineInfoTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *DeadlineInfoTransform) Name() string {
	info := DeadlineInfoTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *DeadlineInfoTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

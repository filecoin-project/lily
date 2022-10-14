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

type FeeDebtTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewFeeDebtTransform(taskName string) *FeeDebtTransform {
	info := miner.FeeDebt{}
	return &FeeDebtTransform{meta: info.Meta(), taskName: taskName}
}

func (s *FeeDebtTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerFeeDebtList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				fd := modeldata.(*miner.FeeDebt)
				sqlModels = append(sqlModels, &minermodel.MinerFeeDebt{
					Height:    int64(fd.Height),
					MinerID:   fd.Miner.String(),
					StateRoot: fd.StateRoot.String(),
					FeeDebt:   fd.Debt.String(),
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

func (s *FeeDebtTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *FeeDebtTransform) Name() string {
	info := FeeDebtTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *FeeDebtTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

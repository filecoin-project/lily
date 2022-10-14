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

type FundsTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewFundsTransform(taskName string) *FundsTransform {
	info := miner.LockedFunds{}
	return &FundsTransform{meta: info.Meta(), taskName: taskName}
}

func (s *FundsTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerLockedFundsList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				lf := modeldata.(*miner.LockedFunds)
				sqlModels = append(sqlModels, &minermodel.MinerLockedFund{
					Height:            int64(lf.Height),
					MinerID:           lf.Miner.String(),
					StateRoot:         lf.StateRoot.String(),
					LockedFunds:       lf.VestingFunds.String(),
					InitialPledge:     lf.InitialPledgeRequirement.String(),
					PreCommitDeposits: lf.PreCommitDeposits.String(),
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

func (s *FundsTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *FundsTransform) Name() string {
	info := FundsTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *FundsTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

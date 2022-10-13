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

type FundsTransform struct {
	meta v2.ModelMeta
}

func NewFundsTransform() *FundsTransform {
	info := miner.LockedFunds{}
	return &FundsTransform{meta: info.Meta()}
}

func (s *FundsTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerLockedFundsList, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
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
				out <- &persistable.Result{Model: sqlModels}
			}
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

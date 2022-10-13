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

type FeeDebtTransform struct {
	meta v2.ModelMeta
}

func NewFeeDebtTransform() *FeeDebtTransform {
	info := miner.FeeDebt{}
	return &FeeDebtTransform{meta: info.Meta()}
}

func (s *FeeDebtTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerFeeDebtList, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
				fd := modeldata.(*miner.FeeDebt)
				sqlModels = append(sqlModels, &minermodel.MinerFeeDebt{
					Height:    int64(fd.Height),
					MinerID:   fd.Miner.String(),
					StateRoot: fd.StateRoot.String(),
					FeeDebt:   fd.Debt.String(),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
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

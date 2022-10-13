package power

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/power"
)

type ClaimedPowerTransform struct {
	meta v2.ModelMeta
}

func NewClaimedPowerTransform() *ClaimedPowerTransform {
	info := power.ClaimedPower{}
	return &ClaimedPowerTransform{meta: info.Meta()}
}

func (s *ClaimedPowerTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(powermodel.PowerActorClaimList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				cp := modeldata.(*power.ClaimedPower)
				if cp.Event == power.Added || cp.Event == power.Modified {
					sqlModels = append(sqlModels, &powermodel.PowerActorClaim{
						Height:          int64(cp.Height),
						MinerID:         cp.Miner.String(),
						StateRoot:       cp.StateRoot.String(),
						RawBytePower:    cp.RawBytePower.String(),
						QualityAdjPower: cp.QualityAdjustedPower.String(),
					})
				}
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *ClaimedPowerTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *ClaimedPowerTransform) Name() string {
	info := ClaimedPowerTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *ClaimedPowerTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

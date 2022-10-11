package power

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/power"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("transform/power")

type ChainPowerTransform struct {
	meta v2.ModelMeta
}

func NewChainPowerTransform() *ChainPowerTransform {
	info := power.ChainPower{}
	return &ChainPowerTransform{meta: info.Meta()}
}

func (s *ChainPowerTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(powermodel.ChainPowerList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				cp := modeldata.(*power.ChainPower)
				sqlModels = append(sqlModels, &powermodel.ChainPower{
					Height:                     int64(cp.Height),
					StateRoot:                  cp.StateRoot.String(),
					TotalRawBytesPower:         cp.TotalRawBytePower.String(),
					TotalQABytesPower:          cp.TotalQualityAdjustedBytePower.String(),
					TotalRawBytesCommitted:     cp.TotalRawBytesCommitted.String(),
					TotalQABytesCommitted:      cp.TotalQualityAdjustedBytesCommitted.String(),
					TotalPledgeCollateral:      cp.TotalPledgeCollateral.String(),
					QASmoothedPositionEstimate: cp.QualityAdjustedSmoothedPositionEstimate.String(),
					QASmoothedVelocityEstimate: cp.QualityAdjustedSmoothedVelocityEstimate.String(),
					MinerCount:                 cp.MinerCount,
					ParticipatingMinerCount:    cp.MinerAboveMinPowerCount,
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *ChainPowerTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *ChainPowerTransform) Name() string {
	info := ChainPowerTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *ChainPowerTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

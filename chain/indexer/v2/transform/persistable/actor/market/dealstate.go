package market

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/market"
)

type DealStateTransformer struct {
	meta v2.ModelMeta
}

func NewDealStateTransformer() *DealStateTransformer {
	info := market.DealState{}
	return &DealStateTransformer{meta: info.Meta()}
}

func (d *DealStateTransformer) ModelType() v2.ModelMeta {
	return d.meta
}

func (d *DealStateTransformer) Name() string {
	info := DealStateTransformer{}
	return reflect.TypeOf(info).Name()
}

func (d *DealStateTransformer) Matcher() string {
	return fmt.Sprintf("^%s$", d.meta.String())
}

func (d *DealStateTransformer) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", d.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(marketmodel.MarketDealStates, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				ds := modeldata.(*market.DealState)
				sqlModels = append(sqlModels, &marketmodel.MarketDealState{
					Height:           int64(ds.Height),
					DealID:           uint64(ds.DealID),
					SectorStartEpoch: int64(ds.SectorStartEpoch),
					LastUpdateEpoch:  int64(ds.LastUpdateEpoch),
					SlashEpoch:       int64(ds.SlashEpoch),
					StateRoot:        ds.StateRoot.String(),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

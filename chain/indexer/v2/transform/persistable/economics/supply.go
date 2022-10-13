package economics

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model"
	chainmodel "github.com/filecoin-project/lily/model/chain"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/economics"
)

type CirculatingSupplyTransform struct {
	meta v2.ModelMeta
}

func NewCirculatingSupplyTransform() *CirculatingSupplyTransform {
	info := economics.ChainEconomics{}
	return &CirculatingSupplyTransform{meta: info.Meta()}
}

func (s *CirculatingSupplyTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(model.PersistableList, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
				m := modeldata.(*economics.ChainEconomics)
				sqlModels = append(sqlModels, &chainmodel.ChainEconomics{
					Height:              int64(m.Height),
					ParentStateRoot:     m.StateRoot.String(),
					CirculatingFil:      m.FilCirculating.String(),
					VestedFil:           m.FilVested.String(),
					MinedFil:            m.FilMined.String(),
					BurntFil:            m.FilBurnt.String(),
					LockedFil:           m.FilLocked.String(),
					FilReserveDisbursed: m.FilReservedDisbursed.String(),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *CirculatingSupplyTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *CirculatingSupplyTransform) Name() string {
	info := CirculatingSupplyTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *CirculatingSupplyTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

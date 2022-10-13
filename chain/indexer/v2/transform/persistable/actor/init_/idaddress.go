package init_

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	initmodel "github.com/filecoin-project/lily/model/actors/init"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/init_"
)

var log = logging.Logger("transform/idaddress")

type IDAddressTransform struct {
	meta v2.ModelMeta
}

func NewIDAddressTransform() *IDAddressTransform {
	state := init_.AddressState{}
	return &IDAddressTransform{meta: state.Meta()}
}

func (i *IDAddressTransform) Name() string {
	return reflect.TypeOf(IDAddressTransform{}).Name()
}

func (i *IDAddressTransform) ModelType() v2.ModelMeta {
	return i.meta
}

func (i *IDAddressTransform) Matcher() string {
	return fmt.Sprintf("^%s$", i.meta.String())
}

func (i *IDAddressTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", i.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("received data", "count", len(res.State().Data))
			sqlModels := make(initmodel.IDAddressList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				m := modeldata.(*init_.AddressState)
				sqlModels = append(sqlModels, &initmodel.IDAddress{
					Height:    int64(m.Height),
					ID:        m.ID.String(),
					Address:   m.Address.String(),
					StateRoot: m.StateRoot.String(),
				})

			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

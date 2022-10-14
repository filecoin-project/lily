package init_

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	initmodel "github.com/filecoin-project/lily/model/actors/init"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/init_"
)

var log = logging.Logger("transform/idaddress")

type IDAddressTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewIDAddressTransform(taskName string) *IDAddressTransform {
	state := init_.AddressState{}
	return &IDAddressTransform{meta: state.Meta(), taskName: taskName}
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

func (i *IDAddressTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", i.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(i.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(initmodel.IDAddressList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				m := modeldata.(*init_.AddressState)
				sqlModels = append(sqlModels, &initmodel.IDAddress{
					Height:    int64(m.Height),
					ID:        m.ID.String(),
					Address:   m.Address.String(),
					StateRoot: m.StateRoot.String(),
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

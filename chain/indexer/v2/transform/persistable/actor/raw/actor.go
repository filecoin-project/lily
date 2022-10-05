package raw

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model/actors/common"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/raw"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("transform/actor")

type ActorTransform struct {
	meta v2.ModelMeta
}

func NewActorTransform() *ActorTransform {
	info := raw.ActorState{}
	return &ActorTransform{meta: info.Meta()}
}

func (a *ActorTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", a.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("received data", "count", len(res.State().Data))
			sqlModels := make(common.ActorList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				m := modeldata.(*raw.ActorState)

				sqlModels = append(sqlModels, &common.Actor{
					Height:    int64(m.Height),
					ID:        m.Address.String(),
					StateRoot: m.StateRoot.String(),
					Code:      m.Code.String(),
					Head:      m.Head.String(),
					Balance:   m.Balance.String(),
					Nonce:     m.Nonce,
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (a *ActorTransform) Name() string {
	return reflect.TypeOf(ActorTransform{}).Name()
}

func (a *ActorTransform) ModelType() v2.ModelMeta {
	return a.meta
}

func (a *ActorTransform) Matcher() string {
	return fmt.Sprintf("^%s$", a.meta.String())
}

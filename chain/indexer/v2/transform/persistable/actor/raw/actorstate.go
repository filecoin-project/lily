package raw

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model/actors/common"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/raw"
)

type ActorStateTransform struct {
	meta v2.ModelMeta
}

func NewActorStateTransform() *ActorStateTransform {
	info := raw.ActorState{}
	return &ActorStateTransform{meta: info.Meta()}
}

func (a *ActorStateTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", a.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("received data", "count", len(res.State().Data))
			sqlModels := make(common.ActorStateList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				m := modeldata.(*raw.ActorState)
				istate, err := vm.DumpActorState(filcns.NewActorRegistry(), &types.Actor{
					Code:    m.Code,
					Head:    m.Head,
					Nonce:   m.Nonce,
					Balance: m.Balance,
				}, m.State)
				if err != nil {
					log.Errorw("dumping actor state", "address", m.Address.String(), "code", builtin.ActorNameByCode(m.Code), "error", err)
					return err
				}
				state, err := json.Marshal(istate)
				if err != nil {
					return err
				}
				sqlModels = append(sqlModels, &common.ActorState{
					Height: int64(m.Height),
					Head:   m.Head.String(),
					Code:   m.Code.String(),
					State:  string(state),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (a *ActorStateTransform) Name() string {
	return reflect.TypeOf(ActorStateTransform{}).Name()
}

func (a *ActorStateTransform) ModelType() v2.ModelMeta {
	return a.meta
}

func (a *ActorStateTransform) Matcher() string {
	return fmt.Sprintf("^%s$", a.meta.String())
}

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
	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actors/common"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/raw"
)

type ActorStateTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewActorStateTransform(taskName string) *ActorStateTransform {
	info := raw.ActorState{}
	return &ActorStateTransform{meta: info.Meta(), taskName: taskName}
}

func (a *ActorStateTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", a.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(a.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(common.ActorStateList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
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
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
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

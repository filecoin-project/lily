package message

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	messages2 "github.com/filecoin-project/lily/model/messages"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
)

type ImplicitParsedMessageTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewImplicitParsedMessageTransform(taskName string) *ImplicitParsedMessageTransform {
	info := messages.VMMessage{}
	return &ImplicitParsedMessageTransform{meta: info.Meta(), taskName: taskName}
}

func (s *ImplicitParsedMessageTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			sqlModels := make(messages2.InternalParsedMessageList, 0, len(res.Models))
			for _, modeldata := range res.Models {
				vm := modeldata.(*messages.VMMessage)
				if !vm.Implicit {
					continue
				}
				params, method, err := util.ParseParams(vm.Params, vm.Method, vm.ToActorCode)
				if err != nil {
					return err
				}
				sqlModels = append(sqlModels, &messages2.InternalParsedMessage{
					Height: int64(vm.Height),
					Cid:    vm.MessageCID.String(),
					From:   vm.From.String(),
					To:     vm.To.String(),
					Value:  vm.Value.String(),
					Method: method,
					Params: params,
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

func (s *ImplicitParsedMessageTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *ImplicitParsedMessageTransform) Name() string {
	info := ImplicitParsedMessageTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *ImplicitParsedMessageTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

package message

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

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

var log = logging.Logger("transform/message")

type VMMessageTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewVMMessageTransform(taskName string) *VMMessageTransform {
	info := messages.VMMessage{}
	return &VMMessageTransform{meta: info.Meta(), taskName: taskName}
}

func (v *VMMessageTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	log.Debugf("run %s", v.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain.ToProcessingReport(v.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			sqlModels := make(messages2.VMMessageList, 0, len(res.Models))
			for _, modeldata := range res.Models {
				m := modeldata.(*messages.VMMessage)
				if m.Implicit {
					continue
				}
				var params string
				var returns string
				var err error
				if m.ToActorCode.Defined() {
					params, _, err = util.ParseParams(m.Params, m.Method, m.ToActorCode)
					if err != nil {
						return err
					}
					if m.ExitCode.IsSuccess() {
						returns, _, err = util.ParseReturn(m.Return, m.Method, m.ToActorCode)
						if err != nil {
							return err
						}
					}
				}
				sqlModels = append(sqlModels, &messages2.VMMessage{
					Height:    int64(m.Height),
					StateRoot: m.StateRoot.String(),
					Cid:       m.MessageCID.String(),
					Source:    m.SourceCID.String(),
					From:      m.From.String(),
					To:        m.To.String(),
					Value:     m.Value.String(),
					Method:    uint64(m.Method),
					ActorCode: m.ToActorCode.String(),
					ExitCode:  int64(m.ExitCode),
					GasUsed:   m.GasUsed,
					Params:    params,
					Returns:   returns,
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

func (v *VMMessageTransform) ModelType() v2.ModelMeta {
	return v.meta
}

func (v *VMMessageTransform) Name() string {
	info := VMMessageTransform{}
	return reflect.TypeOf(info).Name()
}

func (v *VMMessageTransform) Matcher() string {
	return fmt.Sprintf("^%s$", v.meta.String())
}

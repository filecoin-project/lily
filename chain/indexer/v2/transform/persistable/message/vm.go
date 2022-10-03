package message

import (
	"context"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/lens/util"
	messages2 "github.com/filecoin-project/lily/model/messages"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("transform/message")

type VMMessageTransform struct {
	Matcher v2.ModelMeta
}

func NewVMMessageTransform() *VMMessageTransform {
	info := messages.VMMessage{}
	return &VMMessageTransform{Matcher: info.Meta()}
}

func (v *VMMessageTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Info("run VMMessageTransform")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Infow("received data", "count", len(res.State().Data))
			sqlModels := make(messages2.VMMessageList, len(res.State().Data))
			for i, modeldata := range res.State().Data {
				m := modeldata.(*messages.VMMessage)
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
				sqlModels[i] = &messages2.VMMessage{
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
				}
			}
			out <- &persistable.Result{Model: sqlModels}
		}
	}
	return nil
}

func (v *VMMessageTransform) ModelType() v2.ModelMeta {
	return v.Matcher
}

func (v *VMMessageTransform) Name() string {
	info := VMMessageTransform{}
	return reflect.TypeOf(info).Name()
}

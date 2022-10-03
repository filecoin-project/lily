package message

import (
	"context"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	messages2 "github.com/filecoin-project/lily/model/messages"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
	"github.com/filecoin-project/lily/tasks"
)

type MessageTransform struct {
	Matcher v2.ModelMeta
}

func NewMessageTransform() *MessageTransform {
	info := messages.Message{}
	return &MessageTransform{Matcher: info.Meta()}
}

func (v *MessageTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Info("run MessageTransform")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Infow("received data", "count", len(res.State().Data))
			sqlModels := make(messages2.Messages, len(res.State().Data))
			for i, modeldata := range res.State().Data {
				m := modeldata.(*messages.Message)
				sqlModels[i] = &messages2.Message{
					Height:     int64(m.Height),
					Cid:        m.MessageCid.String(),
					From:       m.From.String(),
					To:         m.To.String(),
					Value:      m.Value.String(),
					GasFeeCap:  m.GasFeeCap.String(),
					GasPremium: m.GasPremium.String(),
					GasLimit:   m.GasLimit,
					SizeBytes:  int(m.SizeBytes),
					Nonce:      m.Nonce,
					Method:     uint64(m.Method),
				}
			}
			out <- &persistable.Result{Model: sqlModels}
		}
	}
	return nil
}

func (v *MessageTransform) ModelType() v2.ModelMeta {
	return v.Matcher
}

func (v *MessageTransform) Name() string {
	info := MessageTransform{}
	return reflect.TypeOf(info).Name()
}

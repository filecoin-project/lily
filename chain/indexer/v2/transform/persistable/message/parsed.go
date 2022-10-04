package message

import (
	"context"
	"reflect"

	"github.com/filecoin-project/go-state-types/exitcode"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/lens/util"
	messages2 "github.com/filecoin-project/lily/model/messages"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
	"github.com/filecoin-project/lily/tasks"
)

type ParsedMessageTransform struct {
	Matcher v2.ModelMeta
}

func NewParsedMessageTransform() *ParsedMessageTransform {
	info := messages.ExecutedMessage{}
	return &ParsedMessageTransform{Matcher: info.Meta()}
}

func (p *ParsedMessageTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Info("run ParsedMessageTransform")
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Infow("received data", "count", len(res.State().Data))
			sqlModels := make(messages2.ParsedMessages, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				m := modeldata.(*messages.ExecutedMessage)

				if m.ExitCode == exitcode.ErrSerialization ||
					m.ExitCode == exitcode.ErrIllegalArgument ||
					m.ExitCode == exitcode.SysErrInvalidMethod ||
					// UsrErrUnsupportedMethod TODO: https://github.com/filecoin-project/go-state-types/pull/44
					m.ExitCode == exitcode.ExitCode(22) {
					continue
				}

				var params string
				var method string
				if m.ToActorCode.Defined() {
					var err error
					params, method, err = util.ParseParams(m.Params, m.Method, m.ToActorCode)
					if err != nil {
						return err
					}
				}

				sqlModels = append(sqlModels, &messages2.ParsedMessage{
					Height: int64(m.Height),
					Cid:    m.MessageCid.String(),
					From:   m.From.String(),
					To:     m.To.String(),
					Value:  m.Value.String(),
					Method: method,
					Params: params,
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (p *ParsedMessageTransform) Name() string {
	info := ParsedMessageTransform{}
	return reflect.TypeOf(info).Name()
}

func (p *ParsedMessageTransform) ModelType() v2.ModelMeta {
	return p.Matcher
}

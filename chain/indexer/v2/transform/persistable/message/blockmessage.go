package message

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	messages2 "github.com/filecoin-project/lily/model/messages"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
)

type BlockMessageTransform struct {
	meta v2.ModelMeta
}

func NewBlockMessageTransform() *BlockMessageTransform {
	info := messages.BlockMessage{}
	return &BlockMessageTransform{meta: info.Meta()}
}

func (b *BlockMessageTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", b.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("received data", "count", len(res.State().Data))
			sqlModels := make(messages2.BlockMessages, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				m := modeldata.(*messages.BlockMessage)

				sqlModels = append(sqlModels, &messages2.BlockMessage{
					Height:  int64(m.Height),
					Block:   m.BlockCid.String(),
					Message: m.MessageCid.String(),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (b *BlockMessageTransform) Name() string {
	info := BlockMessageTransform{}
	return reflect.TypeOf(info).Name()
}

func (b *BlockMessageTransform) ModelType() v2.ModelMeta {
	return b.meta
}

func (b *BlockMessageTransform) Matcher() string {
	return fmt.Sprintf("^%s$", b.meta.String())
}

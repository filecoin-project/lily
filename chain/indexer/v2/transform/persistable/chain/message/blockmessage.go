package message

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain"
	"github.com/filecoin-project/lily/model"
	messages2 "github.com/filecoin-project/lily/model/messages"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
)

type BlockMessageTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewBlockMessageTransform(taskName string) *BlockMessageTransform {
	info := messages.BlockMessage{}
	return &BlockMessageTransform{meta: info.Meta(), taskName: taskName}
}

func (b *BlockMessageTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	log.Debugf("run %s", b.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain.ToProcessingReport(b.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			sqlModels := make(messages2.BlockMessages, 0, len(res.Models))
			for _, modeldata := range res.Models {
				m := modeldata.(*messages.BlockMessage)

				sqlModels = append(sqlModels, &messages2.BlockMessage{
					Height:  int64(m.Height),
					Block:   m.BlockCid.String(),
					Message: m.MessageCid.String(),
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

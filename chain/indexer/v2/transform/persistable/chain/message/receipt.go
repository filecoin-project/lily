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

type ReceiptTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewReceiptTransform(taskName string) *ReceiptTransform {
	info := messages.ExecutedMessage{}
	return &ReceiptTransform{meta: info.Meta(), taskName: taskName}
}

func (r *ReceiptTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	log.Debugf("run %s", r.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain.ToProcessingReport(r.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			sqlModels := make(messages2.Receipts, 0, len(res.Models))
			for _, modeldata := range res.Models {
				m := modeldata.(*messages.ExecutedMessage)
				sqlModels = append(sqlModels, &messages2.Receipt{
					Height:    int64(m.Height),
					Message:   m.MessageCid.String(),
					StateRoot: m.StateRoot.String(),
					Idx:       int(m.ReceiptIndex),
					ExitCode:  int64(m.ExitCode),
					GasUsed:   m.GasUsed,
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

func (r *ReceiptTransform) Name() string {
	return reflect.TypeOf(ReceiptTransform{}).Name()
}

func (r *ReceiptTransform) ModelType() v2.ModelMeta {
	return r.meta
}

func (r *ReceiptTransform) Matcher() string {
	return fmt.Sprintf("^%s$", r.meta.String())
}

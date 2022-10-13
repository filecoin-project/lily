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

type ReceiptTransform struct {
	meta v2.ModelMeta
}

func NewReceiptTransform() *ReceiptTransform {
	info := messages.ExecutedMessage{}
	return &ReceiptTransform{meta: info.Meta()}
}

func (r *ReceiptTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", r.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("received data", "count", len(res.State().Data))
			sqlModels := make(messages2.Receipts, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
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
				out <- &persistable.Result{Model: sqlModels}
			}
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

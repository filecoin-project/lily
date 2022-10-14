package block

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/block"
)

var log = logging.Logger("transform/block")

type DrandBlockEntryTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewDrandBlockEntryTransform(taskName string) *DrandBlockEntryTransform {
	info := block.BlockHeader{}
	return &DrandBlockEntryTransform{meta: info.Meta(), taskName: taskName}
}

func (b *DrandBlockEntryTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain.ToProcessingReport(b.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			sqlModels := make(blocks.DrandBlockEntries, 0, len(res.Models))
			for _, modeldata := range res.Models {
				m := modeldata.(*block.BlockHeader)
				for _, ent := range m.BeaconEntries {
					sqlModels = append(sqlModels, &blocks.DrandBlockEntrie{
						Round: ent.Round,
						Block: m.BlockCID.String(),
					})

				}
			}
			if len(sqlModels) > 0 {
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
		}
	}
	return nil
}

func (b *DrandBlockEntryTransform) ModelType() v2.ModelMeta {
	return b.meta
}

func (b *DrandBlockEntryTransform) Name() string {
	info := DrandBlockEntryTransform{}
	return reflect.TypeOf(info).Name()
}

func (b *DrandBlockEntryTransform) Matcher() string {
	return fmt.Sprintf("^%s$", b.meta.String())
}

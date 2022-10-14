package block

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/block"
)

type BlockParentsTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewBlockParentsTransform(taskName string) *BlockParentsTransform {
	info := block.BlockHeader{}
	return &BlockParentsTransform{meta: info.Meta(), taskName: taskName}
}

func (b *BlockParentsTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain.ToProcessingReport(b.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			sqlModels := make(blocks.BlockParents, 0, len(res.Models))
			for _, modeldata := range res.Models {
				m := modeldata.(*block.BlockHeader)
				for _, parent := range m.Parents {
					sqlModels = append(sqlModels, &blocks.BlockParent{
						Height: int64(m.Height),
						Block:  m.BlockCID.String(),
						Parent: parent.String(),
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

func (b *BlockParentsTransform) ModelType() v2.ModelMeta {
	return b.meta
}

func (b *BlockParentsTransform) Name() string {
	info := BlockParentsTransform{}
	return reflect.TypeOf(info).Name()
}

func (b *BlockParentsTransform) Matcher() string {
	return fmt.Sprintf("^%s$", b.meta.String())
}

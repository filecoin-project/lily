package block

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model/blocks"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/block"
	"github.com/filecoin-project/lily/tasks"
)

type BlockParentsTransform struct {
	meta v2.ModelMeta
}

func NewBlockParentsTransform() *BlockParentsTransform {
	info := block.BlockHeader{}
	return &BlockParentsTransform{meta: info.Meta()}
}

func (b *BlockParentsTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(blocks.BlockParents, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
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
				out <- &persistable.Result{Model: sqlModels}
			}
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

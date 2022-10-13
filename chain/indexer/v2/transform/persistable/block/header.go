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
)

type BlockHeaderTransform struct {
	meta v2.ModelMeta
}

func NewBlockHeaderTransform() *BlockHeaderTransform {
	info := block.BlockHeader{}
	return &BlockHeaderTransform{meta: info.Meta()}
}

func (b *BlockHeaderTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(blocks.BlockHeaders, len(res.Models()))
			for i, modeldata := range res.Models() {
				m := modeldata.(*block.BlockHeader)
				sqlModels[i] = &blocks.BlockHeader{
					Height:          int64(m.Height),
					Cid:             m.BlockCID.String(),
					Miner:           m.Miner.String(),
					ParentWeight:    m.ParentWeight.String(),
					ParentBaseFee:   m.ParentBaseFee.String(),
					ParentStateRoot: m.StateRoot.String(),
					WinCount:        m.ElectionProof.WinCount,
					Timestamp:       m.Timestamp,
					ForkSignaling:   m.ForkSignaling,
				}
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (b *BlockHeaderTransform) ModelType() v2.ModelMeta {
	return b.meta
}

func (b *BlockHeaderTransform) Name() string {
	info := BlockHeaderTransform{}
	return reflect.TypeOf(info).Name()
}

func (b *BlockHeaderTransform) Matcher() string {
	return fmt.Sprintf("^%s$", b.meta.String())
}

package blocks

import (
	"context"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"
	"github.com/filecoin-project/lotus/chain/types"
)

func init() {
	model.RegisterTipSetModelExtractor(&blocks.BlockHeader{}, BlockHeaderExtractor{})
	model.RegisterTipSetModelExtractor(&blocks.BlockParent{}, BlockParentExtractor{})
	model.RegisterTipSetModelExtractor(&blocks.DrandBlockEntrie{}, BlockDrandEntriesExtractor{})
}

var _ model.TipSetStateExtractor = (*BlockHeaderExtractor)(nil)

type BlockHeaderExtractor struct{}

func (BlockHeaderExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	var pl model.PersistableList
	for _, bh := range current.Blocks() {
		pl = append(pl, blocks.NewBlockHeader(bh))
	}
	return pl, nil
}

func (BlockHeaderExtractor) Name() string {
	return "block_headers"
}

var _ model.TipSetStateExtractor = (*BlockParentExtractor)(nil)

type BlockParentExtractor struct{}

func (BlockParentExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	var pl model.PersistableList
	for _, bh := range current.Blocks() {
		pl = append(pl, blocks.NewBlockParents(bh))
	}
	return pl, nil
}

func (BlockParentExtractor) Name() string {
	return "blocks_parents"
}

type BlockDrandEntriesExtractor struct{}

func (BlockDrandEntriesExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	var pl model.PersistableList
	for _, bh := range current.Blocks() {
		pl = append(pl, blocks.NewDrandBlockEntries(bh))
	}
	return pl, nil
}

func (BlockDrandEntriesExtractor) Name() string {
	return "drand_block_entries"
}

var _ model.TipSetStateExtractor = (*BlockDrandEntriesExtractor)(nil)

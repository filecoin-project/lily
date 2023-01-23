package fullblock

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"
	"github.com/filecoin-project/lily/pkg/extract/chain"
)

func ExtractBlockHeaders(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) (model.Persistable, error) {
	out := blocks.BlockHeaders{}
	for _, fb := range fullBlocks {
		out = append(out, &blocks.BlockHeader{
			Height:          int64(fb.Block.Height),
			Cid:             fb.Block.Cid().String(),
			Miner:           fb.Block.Miner.String(),
			ParentWeight:    fb.Block.ParentWeight.String(),
			ParentBaseFee:   fb.Block.ParentBaseFee.String(),
			ParentStateRoot: fb.Block.ParentStateRoot.String(),
			WinCount:        fb.Block.ElectionProof.WinCount,
			Timestamp:       fb.Block.Timestamp,
			ForkSignaling:   fb.Block.ForkSignaling,
		})
	}
	return out, nil
}

func ExtractBlockParents(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) (model.Persistable, error) {
	out := blocks.BlockParents{}
	for _, fb := range fullBlocks {
		for _, p := range fb.Block.Parents {
			out = append(out, &blocks.BlockParent{
				Height: int64(fb.Block.Height),
				Block:  fb.Block.Cid().String(),
				Parent: p.String(),
			})
		}
	}
	return out, nil
}

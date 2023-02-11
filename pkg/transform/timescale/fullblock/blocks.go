package fullblock

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lily/pkg/extract/chain"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

func mustMakeTipsetFromFullBlocks(fullBlocks map[cid.Cid]*chain.FullBlock) *types.TipSet {
	var header []*types.BlockHeader
	for _, fb := range fullBlocks {
		header = append(header, fb.Block)
	}
	ts, err := types.NewTipSet(header)
	if err != nil {
		panic(err)
	}
	return ts
}

func ExtractBlockHeaders(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) model.Persistable {
	report := data.StartProcessingReport(tasktype.BlockHeader, mustMakeTipsetFromFullBlocks(fullBlocks))
	for _, fb := range fullBlocks {
		report.AddModels(&blocks.BlockHeader{
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
	return report.Finish()
}

func ExtractBlockParents(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) model.Persistable {
	report := data.StartProcessingReport(tasktype.BlockParent, mustMakeTipsetFromFullBlocks(fullBlocks))
	for _, fb := range fullBlocks {
		for _, p := range fb.Block.Parents {
			report.AddModels(&blocks.BlockParent{
				Height: int64(fb.Block.Height),
				Block:  fb.Block.Cid().String(),
				Parent: p.String(),
			})
		}
	}
	return report.Finish()
}

func ExtractBlockMessages(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) model.Persistable {
	report := data.StartProcessingReport(tasktype.BlockMessage, mustMakeTipsetFromFullBlocks(fullBlocks))
	for _, fb := range fullBlocks {
		for _, msg := range fb.BlsMessages {
			report.AddModels(&messagemodel.BlockMessage{
				Height:  int64(fb.Block.Height),
				Block:   fb.Block.Cid().String(),
				Message: msg.Message.Cid().String(),
			})
		}
		for _, msg := range fb.SecpMessages {
			report.AddModels(&messagemodel.BlockMessage{
				Height:  int64(fb.Block.Height),
				Block:   fb.Block.Cid().String(),
				Message: msg.Message.Cid().String(),
			})
		}
	}
	return report.Finish()
}

package chain

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

type BlockProcessor struct {
}

func NewBlockProcessor() *BlockProcessor {
	return &BlockProcessor{}
}

func (p *BlockProcessor) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.PersistableWithTx, *visormodel.ProcessingReport, error) {
	var pl PersistableWithTxList
	for _, bh := range ts.Blocks() {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		pl = append(pl, blocks.NewBlockHeader(bh))
		pl = append(pl, blocks.NewBlockParents(bh))
		pl = append(pl, blocks.NewDrandBlockEntries(bh))
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	return pl, report, nil
}

func (p *BlockProcessor) Close() error {
	return nil
}

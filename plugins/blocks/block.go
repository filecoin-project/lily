package main

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

var BlockPlugin BlockTaskPlugin

type BlockTaskPlugin struct {
}

func (b *BlockTaskPlugin) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	var pl model.PersistableList
	for _, bh := range ts.Blocks() {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		pl = append(pl, blocks.NewBlockHeader(bh))
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	return pl, report, nil
}

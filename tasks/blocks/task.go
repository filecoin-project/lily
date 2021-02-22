package blocks

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

type Task struct {
}

func NewTask() *Task {
	return &Task{}
}

func (p *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	var pl model.PersistableList
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

func (p *Task) Close() error {
	return nil
}

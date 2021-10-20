package chaineconomics

import (
	"context"

	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

var log = logging.Logger("lily/task/chaineconomics")

type Task struct {
	api tasks.TaskAPI
}

func NewTask(api tasks.TaskAPI) *Task {
	return &Task{
		api: api,
	}
}

func (p *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	ce, err := ExtractChainEconomicsModel(ctx, p.api, ts)
	if err != nil {
		log.Errorw("error received while extracting chain economics, closing lens", "error", err)
		return nil, nil, err
	}

	return ce, report, nil
}

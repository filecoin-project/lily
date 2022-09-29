package v2

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

type TipSetIndexer struct {
	api       tasks.DataSource
	processor *StateExtractor
	tasks     []v2.ModelMeta
	workers   int
}

func NewTipSetIndexer(api tasks.DataSource, tasks []v2.ModelMeta, workers int) *TipSetIndexer {
	return &TipSetIndexer{
		api:       api,
		processor: &StateExtractor{},
		tasks:     tasks,
		workers:   workers,
	}
}

type TipSetResult struct {
	Task     v2.ModelMeta
	Current  *types.TipSet
	Executed *types.TipSet
	Complete bool
	Result   *StateResult
}

func (ti *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) (chan *TipSetResult, error) {
	pts, err := ti.api.TipSet(ctx, ts.Parents())
	if err != nil {
		return nil, err
	}
	// track complete and incomplete tasks for cancellation case
	completedTasks := map[v2.ModelMeta]bool{}
	for _, task := range ti.tasks {
		completedTasks[task] = false
	}

	pool := workerpool.New(ti.workers)
	stateResults := make(chan *StateResult)
	// start processing all the tasks
	if err := ti.processor.Start(ctx, ts, pts, ti.api, pool, ti.tasks, stateResults); err != nil {
		return nil, err
	}

	// complete and incomplete results returned on channel
	outCh := make(chan *TipSetResult, len(ti.tasks))
	go func() {
		// close the outCh when there are no more results to process.
		defer close(outCh)
		for res := range stateResults {
			select {
			case <-ctx.Done():
				for task, complete := range completedTasks {
					if complete {
						continue
					}
					outCh <- &TipSetResult{
						Task:     task,
						Current:  ts,
						Executed: pts,
						Complete: false,
						Result:   nil,
					}
				}
				return
			default:
				completedTasks[res.Task] = true
				outCh <- &TipSetResult{
					Task:     res.Task,
					Current:  ts,
					Executed: pts,
					Complete: true,
					Result:   res,
				}
			}
		}
	}()
	return outCh, nil
}

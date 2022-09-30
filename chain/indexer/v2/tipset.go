package v2

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

type TipSetIndexer struct {
	api       tasks.DataSource
	processor *extract.StateExtractor
	tasks     []v2.ModelMeta
	workers   int
}

func NewTipSetIndexer(api tasks.DataSource, tasks []v2.ModelMeta, workers int) *TipSetIndexer {
	return &TipSetIndexer{
		api:       api,
		processor: &extract.StateExtractor{},
		tasks:     tasks,
		workers:   workers,
	}
}

type TipSetResult struct {
	task     v2.ModelMeta
	current  *types.TipSet
	executed *types.TipSet
	complete bool
	result   *extract.StateResult
}

func (t *TipSetResult) Task() v2.ModelMeta {
	return t.task
}

func (t *TipSetResult) Current() *types.TipSet {
	return t.current
}

func (t *TipSetResult) Executed() *types.TipSet {
	return t.executed
}

func (t *TipSetResult) Complete() bool {
	return t.complete
}

func (t *TipSetResult) State() *extract.StateResult {
	return t.result
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
	stateResults := make(chan *extract.StateResult)
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
						task:     task,
						current:  ts,
						executed: pts,
						complete: false,
						result:   nil,
					}
				}
				return
			default:
				completedTasks[res.Task] = true
				outCh <- &TipSetResult{
					task:     res.Task,
					current:  ts,
					executed: pts,
					complete: true,
					result:   res,
				}
			}
		}
	}()
	return outCh, nil
}

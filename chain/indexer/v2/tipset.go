package v2

import (
	"context"
	"sync"

	"github.com/filecoin-project/lotus/chain/types"

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

func NewTipSetIndexer(api tasks.DataSource, tasks []v2.ModelMeta, workers int) (*TipSetIndexer, error) {
	processor, err := extract.NewStateExtractor(api, tasks, workers, workers, workers)
	if err != nil {
		return nil, err
	}
	return &TipSetIndexer{
		api:       api,
		processor: processor,
		tasks:     tasks,
		workers:   workers,
	}, nil
}

type TipSetResult struct {
	task            v2.ModelMeta
	current         *types.TipSet
	executed        *types.TipSet
	complete        bool
	result          *extract.StateResult
	models          []v2.LilyModel
	extractionState interface{}
}

func (t *TipSetResult) ExtractionState() interface{} {
	return t.extractionState
}

func (t *TipSetResult) Models() []v2.LilyModel {
	return t.models
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

func (ti *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) (chan *TipSetResult, error) {
	outCh := make(chan *TipSetResult, len(ti.tasks))

	pts, err := ti.api.TipSet(ctx, ts.Parents())
	if err != nil {
		return nil, err
	}
	// track complete and incomplete tasks for cancellation case
	completedTasks := map[v2.ModelMeta]bool{}
	for _, task := range ti.tasks {
		completedTasks[task] = false
	}

	// start processing all the tasks
	tsCh, actCh, errCh := ti.processor.Start(ctx, ts, pts)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for res := range tsCh {
			outCh <- &TipSetResult{
				task:            res.Task,
				current:         ts,
				executed:        pts,
				complete:        true,
				models:          res.Models,
				extractionState: res,
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for res := range actCh {
			outCh <- &TipSetResult{
				task:            res.Task,
				current:         ts,
				executed:        pts,
				complete:        true,
				models:          res.Results.Models(),
				extractionState: res,
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for res := range errCh {
			log.Errorw("TODO FORREST HANDLE ERROR CHANNEL", "error", res.Error())
		}
	}()

	go func() {
		wg.Wait()
		close(outCh)
	}()

	return outCh, nil
}

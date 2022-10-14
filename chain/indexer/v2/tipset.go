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

func (ti *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet, pts *types.TipSet) (chan *TipSetResult, error) {
	outCh := make(chan *TipSetResult, len(ti.tasks))
	errCh := make(chan error, 1)

	tsCh, actCh, err := ti.processor.Start(ctx, ts, pts)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for res := range tsCh {
			completed := true
			if res.Error != nil {
				completed = false
			}
			outCh <- &TipSetResult{
				task:            res.Task,
				current:         ts,
				executed:        pts,
				complete:        completed,
				models:          res.Models,
				extractionState: res,
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for res := range actCh {
			completed := true
			actErrors := res.Results.Errors()
			if len(actErrors) > 0 {
				completed = false
			}
			outCh <- &TipSetResult{
				task:            res.Task,
				current:         ts,
				executed:        pts,
				complete:        completed,
				models:          res.Results.Models(),
				extractionState: res,
			}
		}
	}()

	go func() {
		wg.Wait()
		close(outCh)
		close(errCh)
	}()

	return outCh, nil
}

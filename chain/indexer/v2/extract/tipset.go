package extract

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

type TipSetStateResult struct {
	Task      v2.ModelMeta
	TipSet    *types.TipSet
	StartTime time.Time
	Duration  time.Duration
	Models    []v2.LilyModel
	Error     *TipSetExtractorError
}

type TipSetExtractorError struct {
	Error error
}

func TipSetState(ctx context.Context, workers int, api tasks.DataSource, current, executed *types.TipSet, extractors map[v2.ModelMeta]v2.ExtractorFn, results chan *TipSetStateResult) {
	pool := workerpool.New(workers)
	for task, extractor := range extractors {
		task := task
		extractor := extractor
		pool.Submit(func() {
			select {
			case <-ctx.Done():
				return
			default:
				start := time.Now()
				data, err := extractor(ctx, api, current, executed)
				duration := time.Since(start)
				log.Debugw("extracted model", "type", task.String(), "duration", time.Since(start))
				out := &TipSetStateResult{
					Task:      task,
					TipSet:    current,
					StartTime: start,
					Duration:  duration,
					Models:    data,
				}
				if err != nil {
					out.Error = &TipSetExtractorError{Error: err}
				}
				results <- out
				log.Infow("completed extraction for tipset", "task", task.String(), "duration", duration, "count", len(data))
			}
		})
	}
	pool.StopWait()
}

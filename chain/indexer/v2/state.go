package v2

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("v2_indexer")

type StateExtractor struct {
}

type StateResult struct {
	Task      v2.ModelMeta
	Error     error
	Data      []v2.LilyModel
	StartedAt time.Time
	Duration  time.Duration
}

func (se *StateExtractor) Start(ctx context.Context, current, executed *types.TipSet, api tasks.DataSource, pool *workerpool.WorkerPool, tasks []v2.ModelMeta, results chan *StateResult) error {
	tsTaskMap := map[v2.ModelMeta]v2.ExtractorFn{}
	actTaskMap := map[v2.ModelMeta]v2.ActorExtractorFn{}
	for _, task := range tasks {
		switch task.Kind {
		case v2.ModelActorKind:
			efn, err := v2.LookupActorExtractor(task)
			if err != nil {
				return err
			}
			actTaskMap[task] = efn
		case v2.ModelTsKind:
			efn, err := v2.LookupExtractor(task)
			if err != nil {
				return err
			}
			tsTaskMap[task] = efn
		default:
			panic("developer error")
		}
	}

	for task, extractor := range tsTaskMap {
		task := task
		extractor := extractor
		pool.Submit(func() {
			select {
			case <-ctx.Done():
				return
			default:
				start := time.Now()
				data, err := extractor(ctx, api, current, executed)
				results <- &StateResult{
					Task:      task,
					Error:     err,
					Data:      data,
					StartedAt: start,
					Duration:  time.Since(start),
				}
			}
		})
	}
	changes, err := api.ActorStateChanges(ctx, current, executed)
	if err != nil {
		return err
	}

	codeToActors := make(map[cid.Cid][]actorstate.ActorInfo)
	for addr, change := range changes {
		codeToActors[change.Actor.Code] = append(codeToActors[change.Actor.Code], actorstate.ActorInfo{
			Actor:      change.Actor,
			ChangeType: change.ChangeType,
			Address:    addr,
			Current:    current,
			Executed:   executed,
		})
	}

	for task, extractor := range actTaskMap {
		task := task
		extractor := extractor
		supportedActors, err := v2.LookupActorTypeThing(task)
		if err != nil {
			return err
		}
		var actorsForExtractor []actorstate.ActorInfo
		if err := supportedActors.ForEach(func(c cid.Cid) error {
			a, ok := codeToActors[c]
			if !ok {
				return nil
			}
			actorsForExtractor = append(actorsForExtractor, a...)
			return nil
		}); err != nil {
			return err
		}
		for _, act := range actorsForExtractor {
			act := act
			pool.Submit(func() {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()
					data, err := extractor(ctx, api, current, executed, act)
					results <- &StateResult{
						Task:      task,
						Error:     err,
						Data:      data,
						StartedAt: start,
						Duration:  time.Since(start),
					}
				}
			})
		}
	}
	go func() {
		pool.StopWait()
		close(results)
	}()
	return nil
}

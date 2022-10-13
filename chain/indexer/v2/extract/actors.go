package extract

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type ActorStateResult struct {
	Task      v2.ModelMeta
	TipSet    *types.TipSet
	Results   ActorExtractorResultList
	StartTime time.Time
	Duration  time.Duration
}

type ActorExtractorResult struct {
	Info      actorstate.ActorInfo
	StartTime time.Time
	Duration  time.Duration
	Models    []v2.LilyModel
	Error     *ActorExtractorError
}

type ActorExtractorResultList []*ActorExtractorResult

func (l ActorExtractorResultList) Models() []v2.LilyModel {
	var out []v2.LilyModel
	for _, res := range l {
		out = append(out, res.Models...)
	}
	return out
}

type ActorExtractorError struct {
	Error error
}

func ActorStates(ctx context.Context, workers int, extractorWorkers int, api tasks.DataSource, current, executed *types.TipSet, actors map[v2.ModelMeta]v2.ActorExtractorFn, results chan *ActorStateResult) error {
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
	pool := workerpool.New(workers)
	for task, extractor := range actors {
		// due to parallel call below
		task := task
		extractor := extractor

		// get list of supported actor codes for this task type
		supportedActors, err := v2.LookupActorTypeThing(task)
		if err != nil {
			return err
		}
		// from the set of changes actors filter for actor codes supported by this extractor
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

		// execute extractor in parallel
		pool.Submit(func() {
			res := runActorExtractors(ctx, extractorWorkers, task, api, current, executed, extractor, actorsForExtractor)
			results <- res
		})
	}
	// wait for all extractors to complete
	pool.StopWait()

	return nil
}

func runActorExtractors(ctx context.Context, workers int, task v2.ModelMeta, api tasks.DataSource, current, executed *types.TipSet, extractFn v2.ActorExtractorFn, candidates []actorstate.ActorInfo) *ActorStateResult {
	results := make([]*ActorExtractorResult, len(candidates))
	actorsStart := time.Now()
	pool := workerpool.New(workers)

	// start an extractor for each candidate actor and collect its result in parallel.
	for i, candidate := range candidates {
		// due to parallel call
		actor := candidate
		i := i
		pool.Submit(func() {
			extractorStart := time.Now()
			models, err := extractFn(ctx, api, current, executed, actor)
			results[i] = &ActorExtractorResult{
				Info:      actor,
				StartTime: extractorStart,
				Duration:  time.Since(extractorStart),
				Models:    models,
			}
			if err != nil {
				results[i].Error = &ActorExtractorError{Error: err}
				log.Errorw("actor extractor", "info", zap.Inline(actor), "error", err)
			}
		})
	}

	// wait for all parallel extraction to complete
	pool.StopWait()

	// return result of extraction for candidates.
	out := &ActorStateResult{
		Task:      task,
		TipSet:    current,
		Results:   results,
		StartTime: actorsStart,
		Duration:  time.Since(actorsStart),
	}
	log.Infow("completed extraction for actors", "task", task.String(), "duration", out.Duration, "count", len(out.Results))
	return out
}

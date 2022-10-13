package extract

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract")

func NewStateExtractor(api tasks.DataSource, tasks []v2.ModelMeta, tsWorkers, actorWorkers, actorExctractorWorkers int) (*StateExtractor, error) {
	tsTaskMap := map[v2.ModelMeta]v2.ExtractorFn{}
	actTaskMap := map[v2.ModelMeta]v2.ActorExtractorFn{}
	for _, task := range tasks {
		switch task.Kind {
		case v2.ModelActorKind:
			efn, err := v2.LookupActorExtractor(task)
			if err != nil {
				return nil, err
			}
			actTaskMap[task] = efn
		case v2.ModelTsKind:
			efn, err := v2.LookupExtractor(task)
			if err != nil {
				return nil, err
			}
			tsTaskMap[task] = efn
		default:
			panic("developer error")
		}
	}
	return &StateExtractor{
		api:                   api,
		tipsetTasks:           tsTaskMap,
		actorTasks:            actTaskMap,
		TipSetTaskWorkers:     tsWorkers,
		ActorTaskWorkers:      actorWorkers,
		ActorExtractorWorkers: actorExctractorWorkers,
	}, nil
}

type StateExtractor struct {
	api                   tasks.DataSource
	tipsetTasks           map[v2.ModelMeta]v2.ExtractorFn
	actorTasks            map[v2.ModelMeta]v2.ActorExtractorFn
	TipSetTaskWorkers     int
	ActorTaskWorkers      int
	ActorExtractorWorkers int
}

type StateResult struct {
	Task      v2.ModelMeta
	Error     error
	Data      []v2.LilyModel
	StartedAt time.Time
	Duration  time.Duration
}

func (se *StateExtractor) Start(ctx context.Context, current, executed *types.TipSet) (chan *TipSetStateResult, chan *ActorStateResult, chan error) {
	// todo maybe buffer these, or add a config for it
	tipsetsCh := make(chan *TipSetStateResult, len(se.tipsetTasks))
	actorsCh := make(chan *ActorStateResult, len(se.actorTasks))
	errorCh := make(chan error)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := TipSetState(ctx, se.TipSetTaskWorkers, se.api, current, executed, se.tipsetTasks, tipsetsCh); err != nil {
			log.Errorw("tipset state extraction", "error", err)
			errorCh <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ActorStates(ctx, se.ActorTaskWorkers, se.ActorExtractorWorkers, se.api, current, executed, se.actorTasks, actorsCh); err != nil {
			log.Errorw("actor state extraction", "error", err)
			errorCh <- err
		}
	}()

	go func() {
		wg.Wait()
		close(tipsetsCh)
		close(actorsCh)
		close(errorCh)
	}()

	return tipsetsCh, actorsCh, errorCh
}

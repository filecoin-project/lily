package network

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/tasks/survey/peeragents"
)

const (
	PeerAgentsTask = "peeragents" // task that observes connected peer agents
)

var log = logging.Logger("lily/network")

type API interface {
	peeragents.API
}

func NewSurveyer(api API, storage model.Storage, interval time.Duration, name string, tasks []string) (*Surveyer, error) {
	if interval <= 0 {
		return nil, xerrors.Errorf("surveyer interval must be greater than zero: %d", interval)
	}

	obs := &Surveyer{
		interval: interval,
		storage:  storage,
		name:     name,
		tasks:    map[string]Task{},
	}

	for _, task := range tasks {
		switch task {
		case PeerAgentsTask:
			obs.tasks[PeerAgentsTask] = peeragents.NewTask(api)
		default:
			return nil, xerrors.Errorf("unknown task: %s", task)
		}
	}

	return obs, nil
}

// An Surveyer observes features of the filecoin network
type Surveyer struct {
	interval time.Duration
	storage  model.Storage
	name     string
	tasks    map[string]Task
	done     chan struct{}
}

func (s *Surveyer) Close() {
	return
}

// Run starts observing the filecoin netwoirk and continues until the context is done or
// a fatal error occurs.
func (s *Surveyer) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	s.done = make(chan struct{})
	defer close(s.done)

	// Perform an initial tick before waiting
	s.Tick(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

func (s *Surveyer) Done() <-chan struct{} {
	return s.done
}

func (s *Surveyer) Details() (string, map[string]interface{}) {
	return "surveyer", map[string]interface{}{
		"name":     s.name,
		"interval": s.interval,
	}
}

// Tick is called on each tick of the surveyer's interval
func (s *Surveyer) Tick(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "Surveyer.Tick")
	defer span.End()

	// Do as much indexing as possible in the specified time interval
	tctx, cancel := context.WithTimeout(ctx, s.interval)
	defer cancel()

	inFlight := 0
	results := make(chan *TaskResult, len(s.tasks))

	// Run each tipset processing task concurrently
	for name, task := range s.tasks {
		inFlight++
		go s.runTask(tctx, task, name, results)
	}

	// Wait for all tasks to complete
	for inFlight > 0 {
		var res *TaskResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res = <-results:
		}
		inFlight--

		llt := log.With("task", res.Task)

		// Was there a fatal error?
		if res.Error != nil {
			llt.Errorw("task returned with error", "error", res.Error.Error())
			return res.Error
		}

		llt.Debugw("task report", "time", res.CompletedAt.Sub(res.StartedAt))

		startPersist := time.Now()
		if err := s.storage.PersistBatch(ctx, res.Data); err != nil {
			stats.Record(ctx, metrics.PersistFailure.M(1))
			llt.Errorw("persistence failed", "error", err)
		} else {
			llt.Debugw("task data persisted", "time", time.Since(startPersist))
		}

	}

	return nil
}

func (s *Surveyer) runTask(ctx context.Context, task Task, name string, results chan *TaskResult) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	data, err := task.Process(ctx)
	if err != nil {
		stats.Record(ctx, metrics.ProcessingFailure.M(1))
		results <- &TaskResult{
			Task:        name,
			Error:       err,
			StartedAt:   start,
			CompletedAt: time.Now(),
		}
		return
	}
	results <- &TaskResult{
		Task:        name,
		Data:        data,
		StartedAt:   start,
		CompletedAt: time.Now(),
	}
}

// A TaskResult is either some data to persist or an error which indicates that the task did not complete. Partial
// completions are possible provided the Data contains a persistable log of the results.
type TaskResult struct {
	Task        string
	Error       error
	Data        model.Persistable
	StartedAt   time.Time
	CompletedAt time.Time
}

type Task interface {
	Process(ctx context.Context) (model.Persistable, error)
	Close() error
}

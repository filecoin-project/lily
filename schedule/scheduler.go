package schedule

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/node/modules/helpers"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
)

var log = logging.Logger("schedule")

type Job interface {
	// Run starts running the task and blocks until the context is done or
	// an error occurs.
	Run(context.Context) error
}

type JobConfig struct {
	// ID of the task
	id JobID

	// to cancel the task
	cancel context.CancelFunc

	// running is true if the job is executing, false otherwise.
	running bool

	log *zap.SugaredLogger

	// Name is a human readable name for the job for use in logging
	Name string

	// Job is the job that will be executed.
	Job Job

	// Locker is an optional lock that must be taken before the job can execute.
	Locker Locker

	// RestartOnFailure controls whether the job should be restarted if it stops with an error.
	RestartOnFailure bool

	// RestartOnCompletion controls whether the job should be restarted if it stops without an error.
	RestartOnCompletion bool

	// RestartDelay is the amount of time to wait before restarting a stopped job
	RestartDelay time.Duration
}

// Locker represents a general lock that a job may need to take before operating.
type Locker interface {
	Lock(context.Context) error
	Unlock(context.Context) error
}

func NewScheduler(jobDelay time.Duration, scheduledJobs ...*JobConfig) *Scheduler {
	// Enforce a minimum delay
	if jobDelay == 0 {
		jobDelay = 100 * time.Millisecond
	}
	s := &Scheduler{
		jobID:    0,
		jobDelay: jobDelay,
		jobQueue: make(chan *JobConfig),
		jobs:     make(map[JobID]*JobConfig),

		scheduledJobComplete: make(chan struct{}, len(scheduledJobs)),
		scheduledJobsRunning: len(scheduledJobs),

		workerJobComplete: make(chan struct{}),
		workerJobsRunning: 0,
	}

	// scheduled jobs added here will be started when Scheduler.Run is called.
	for _, st := range scheduledJobs {
		s.jobID++
		st.id = s.jobID
		st.log = log.With("id", st.id, "name", st.Name)
		s.jobs[s.jobID] = st
	}
	return s
}

func NewSchedulerDaemon(mctx helpers.MetricsCtx, lc fx.Lifecycle) *Scheduler {
	s := NewScheduler(0)
	go func() {
		if err := s.Run(mctx); err != nil {
			log.Errorw("Scheduler Stopped", "error", err)
		}
	}()

	return s
}

type Scheduler struct {
	jobs   map[JobID]*JobConfig
	jobID  JobID
	jobsMu sync.Mutex

	jobDelay time.Duration

	context context.Context

	jobQueue chan *JobConfig

	scheduledJobComplete chan struct{}
	scheduledJobsRunning int

	workerJobComplete chan struct{}
	workerJobsRunning int
}

func (s *Scheduler) Submit(tc *JobConfig) JobID {
	s.jobsMu.Lock()
	s.jobID++
	tc.id = s.jobID
	s.jobsMu.Unlock()

	s.jobQueue <- tc
	return s.jobID
}

// Run starts running the scheduler and blocks until the context is done.
func (s *Scheduler) Run(ctx context.Context) error {
	log.Info("Starting Scheduler")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// used as context for jobs submitted, ensure they are canceled when context is canceled.
	s.context = ctx

	s.jobsMu.Lock()
	for _, tc := range s.jobs {
		go s.execute(tc, s.scheduledJobComplete)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// A little jitter between scheduledTasks to reduce thundering herd effects on api.
		wait.SleepWithJitter(s.jobDelay, 2)
	}
	s.jobsMu.Unlock()

	// Wait until the context is done and handle new jobs as they are submitted.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case newTask := <-s.jobQueue:
			s.jobsMu.Lock()
			s.jobs[newTask.id] = newTask
			s.jobsMu.Unlock()

			newTask.log = log.With("id", newTask.id, "name", newTask.Name)
			newTask.log.Infow("new job received")

			go s.execute(newTask, s.workerJobComplete)
		case <-s.scheduledJobComplete:
			// A job has completed
			s.scheduledJobsRunning--
			if s.scheduledJobsRunning == 0 {
				log.Info("no scheduled jobs running")
			}
		case <-s.workerJobComplete:
			s.workerJobsRunning--
			if s.workerJobsRunning == 0 {
				log.Info("no worker jobs running")
			}
		}
	}
}

func (s *Scheduler) StartJob(id JobID) error {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return xerrors.Errorf("starting worker ID: %d not found", id)
	}

	if job.running {
		return xerrors.Errorf("starting worker ID: %d already running", id)
	}
	job.log.Info("starting job")
	go s.execute(job, s.workerJobComplete)
	return nil
}

func (s *Scheduler) StopJob(id JobID) error {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return xerrors.Errorf("starting worker ID: %d not found", id)
	}

	if !job.running {
		return xerrors.Errorf("starting worker ID: %d already running", id)
	}

	job.log.Info("stopping job")
	job.cancel()
	job.running = false
	return nil
}

type JobResult struct {
	ID                  JobID
	Running             bool
	Name                string
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
}

var InvalidJobID = JobID(0)

type JobID int

func (s *Scheduler) Jobs() []JobResult {
	if len(s.jobs) == 0 {
		return nil
	}
	var out []JobResult
	for _, j := range s.jobs {
		out = append(out, JobResult{
			ID:                  j.id,
			Running:             j.running,
			Name:                j.Name,
			RestartOnFailure:    j.RestartOnFailure,
			RestartOnCompletion: j.RestartOnCompletion,
			RestartDelay:        j.RestartDelay,
		})
	}
	return out
}

func (s *Scheduler) execute(tc *JobConfig, complete chan struct{}) {
	ctx, cancel := context.WithCancel(s.context)
	tc.cancel = cancel
	tc.running = true
	// Report job is complete when this goroutine exits
	defer func() {
		complete <- struct{}{}
		tc.running = false
		tc.cancel()
		tc.log.Info("job execution ended")
	}()

	// Attempt to get the job lock if specified
	// TODO: can this be removed? I don't think we use it, maybe we will later?
	if tc.Locker != nil {
		if err := tc.Locker.Lock(ctx); err != nil {
			if errors.Is(err, storage.ErrLockNotAcquired) {
				tc.log.Infow("job not started: lock not acquired")
				return
			}
			tc.log.Errorw("job not started: lock not acquired", "error", err.Error())
			return
		}
		defer func() {
			if err := tc.Locker.Unlock(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					tc.log.Errorw("failed to unlock job", "error", err.Error())
				}
			}
		}()
	}

	// Keep this job running forever
	doneFirstRun := false
	for {

		// Is the context done?
		select {
		case <-ctx.Done():
			return
		default:
		}

		if doneFirstRun {
			tc.log.Infow("restarting job", "delay", tc.RestartDelay)
			if tc.RestartDelay > 0 {
				time.Sleep(tc.RestartDelay)
			}
		} else {
			tc.log.Info("running job")
			doneFirstRun = true
		}

		err := tc.Job.Run(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			tc.log.Errorw("job exited with failure", "error", err.Error())
			// TODO here is where we can add the error to some kind of job status struct

			if !tc.RestartOnFailure {
				// Exit the job
				break
			}
		} else {
			tc.log.Info("job exited cleanly")

			if !tc.RestartOnCompletion {
				// Exit the job
				break
			}
		}
	}
}

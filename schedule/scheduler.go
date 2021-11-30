package schedule

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/node/modules/helpers"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lily/wait"
)

var log = logging.Logger("lily/schedule")

type Job interface {
	// Run starts running the task and blocks until the context is done or
	// an error occurs. Run may be called again after an error or timeout to
	// retry the job so implemententions must ensure that Run resets any
	// necessary state.
	Run(context.Context) error
	Done() <-chan struct{}
}

type JobConfig struct {
	lk sync.Mutex
	// ID of the task
	id JobID

	// to cancel the task
	cancel context.CancelFunc

	// running is true if the job is executing, false otherwise.
	running bool

	// errorMsg will contain a (helpful) string iff a jobs execution has halted due to an error.
	errorMsg string

	log *zap.SugaredLogger

	// Name is a human readable name for the job for use in logging
	Name string

	// Tasks is a list of tasks the job performs
	Tasks []string

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

	// Type is a human readable type for the job for use in logging.
	Type string

	// Params is a map of additional parameters that add human readable context to the job.
	Params map[string]string

	// StartedAt is the time the job started running.
	StartedAt time.Time

	// EndedAt is the time the job stopped running, either through successful completion or failure. Reset if job is restarted.
	EndedAt time.Time
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

		daemonMode: false,
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
	s.daemonMode = true

	ctx, cancel := context.WithCancel(mctx)
	go func() {
		if err := s.Run(ctx); err != nil {
			if err != context.Canceled {
				log.Errorw("Scheduler Stopped", "error", err)
			}
			log.Infow("Scheduler Stopper", "error", err)
		}
	}()
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
	return s
}

type Scheduler struct {
	jobs   map[JobID]*JobConfig
	jobsMu sync.Mutex

	jobID   JobID
	jobIDMu sync.Mutex

	jobDelay time.Duration

	context context.Context

	jobQueue chan *JobConfig

	scheduledJobComplete chan struct{}
	scheduledJobsRunning int

	workerJobComplete chan struct{}
	workerJobsRunning int

	// if daemonMode is set to true the scheduler will continue to run until its context is canceled.
	// else the scheduler will exit when all scheduled jobs are complete.
	daemonMode bool
}

type JobSubmitResult struct {
	ID                  JobID
	Name                string
	Type                string
	Tasks               []string
	Params              map[string]string
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
}

func (s *Scheduler) Submit(jc *JobConfig) *JobSubmitResult {
	s.jobIDMu.Lock()
	defer s.jobIDMu.Unlock()

	s.jobID++
	jc.id = s.jobID
	s.jobQueue <- jc

	return &JobSubmitResult{
		ID:                  jc.id,
		Name:                jc.Name,
		Type:                jc.Type,
		Tasks:               jc.Tasks,
		Params:              jc.Params,
		RestartOnFailure:    jc.RestartOnFailure,
		RestartOnCompletion: jc.RestartOnCompletion,
		RestartDelay:        jc.RestartDelay,
	}
}

// Run starts running the scheduler and blocks until the context is done.
func (s *Scheduler) Run(ctx context.Context) error {
	log.Info("Starting Scheduler")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// used as context for jobs submitted, ensure they are canceled when context is canceled.
	s.context = ctx

	// we don't lock here since jobs can only be written to in the for loop following this.
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

	// Wait until the context is done and handle new jobs as they are submitted.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case newTask := <-s.jobQueue:
			s.jobsMu.Lock()

			s.jobs[newTask.id] = newTask
			newTask.log = log.With("id", newTask.id, "name", newTask.Name)
			newTask.log.Infow("new job received")

			s.jobsMu.Unlock()

			go s.execute(newTask, s.workerJobComplete)
		case <-s.scheduledJobComplete:
			// A job has completed
			s.scheduledJobsRunning--
			if s.scheduledJobsRunning == 0 {
				log.Info("no scheduled jobs running")
				if !s.daemonMode {
					log.Info("all scheduled jobs complete, scheduler exiting")
					return nil
				}
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
	job, err := s.getJob(id)
	if err != nil {
		return xerrors.Errorf("start job: %w", err)
	}

	job.lk.Lock()
	// clear any error messages if we are starting the job
	job.errorMsg = ""
	if job.running {
		job.lk.Unlock()
		return xerrors.Errorf("starting worker ID: %d already running", id)
	}
	job.lk.Unlock()

	job.log.Info("starting job")
	go s.execute(job, s.workerJobComplete)
	return nil
}

func (s *Scheduler) StopJob(id JobID) error {
	job, err := s.getJob(id)
	if err != nil {
		return xerrors.Errorf("stop job: %w", err)
	}

	job.lk.Lock()
	defer job.lk.Unlock()

	if !job.running {
		return xerrors.Errorf("stopping job ID: %d already stopped", id)
	}

	job.log.Infow("stopping job", "id", job)
	job.cancel()
	return nil
}

func (s *Scheduler) WaitJob(id JobID) (*JobListResult, error) {
	job, err := s.getJob(id)
	if err != nil {
		return nil, xerrors.Errorf("wait job: %w", err)
	}

	// wait on the job to complete
	<-job.Job.Done()

	// fetch the job to get the latest results (EndedAt and Running will have changed)
	job, err = s.getJob(id)
	if err != nil {
		return nil, xerrors.Errorf("wait job: %w", err)
	}
	return &JobListResult{
		ID:                  job.id,
		Name:                job.Name,
		Type:                job.Type,
		Error:               job.errorMsg,
		Tasks:               job.Tasks,
		Running:             job.running,
		RestartOnFailure:    job.RestartOnFailure,
		RestartOnCompletion: job.RestartOnCompletion,
		RestartDelay:        job.RestartDelay,
		Params:              job.Params,
		StartedAt:           job.StartedAt,
		EndedAt:             job.EndedAt,
	}, nil
}

func (s *Scheduler) getJob(id JobID) (*JobConfig, error) {
	s.jobsMu.Lock()
	job, ok := s.jobs[id]
	s.jobsMu.Unlock()
	if !ok {
		return nil, xerrors.Errorf("job id: %d not found", id)
	}
	return job, nil
}

type JobListResult struct {
	ID    JobID
	Name  string
	Type  string
	Error string
	Tasks []string

	Running bool

	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration

	Params    map[string]string
	StartedAt time.Time
	EndedAt   time.Time
}

var InvalidJobID = JobID(0)

type JobID int

func (s *Scheduler) Jobs() []JobListResult {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	if len(s.jobs) == 0 {
		return nil
	}
	var out []JobListResult
	for _, j := range s.jobs {
		j.lk.Lock()
		out = append(out, JobListResult{
			ID:                  j.id,
			Name:                j.Name,
			Tasks:               j.Tasks,
			Type:                j.Type,
			Error:               j.errorMsg,
			Running:             j.running,
			RestartOnFailure:    j.RestartOnFailure,
			RestartOnCompletion: j.RestartOnCompletion,
			RestartDelay:        j.RestartDelay,
			Params:              j.Params,
			StartedAt:           j.StartedAt,
			EndedAt:             j.EndedAt,
		})
		j.lk.Unlock()
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *Scheduler) execute(jc *JobConfig, complete chan struct{}) {
	ctx, cancel := context.WithCancel(s.context)
	ctx = metrics.WithTagValue(ctx, metrics.Job, jc.Name)

	jc.lk.Lock()
	jc.cancel = cancel
	jc.running = true
	jc.StartedAt = time.Now().UTC()
	jc.EndedAt = time.Time{}
	jc.lk.Unlock()

	// Report job is complete when this goroutine exits
	defer func() {
		complete <- struct{}{}

		jc.lk.Lock()
		jc.running = false
		jc.EndedAt = time.Now().UTC()
		jc.cancel()
		jc.lk.Unlock()

		jc.log.Info("job execution ended")
	}()

	// Attempt to get the job lock if specified
	if jc.Locker != nil {
		if err := jc.Locker.Lock(ctx); err != nil {
			jc.errorMsg = err.Error()
			if errors.Is(err, storage.ErrLockNotAcquired) {
				jc.log.Infow("job not started: lock not acquired")
				return
			}
			jc.log.Errorw("job not started: lock not acquired", "error", err.Error())
			return
		}
		defer func() {
			if err := jc.Locker.Unlock(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					jc.errorMsg = err.Error()
					jc.log.Errorw("failed to unlock job", "error", err.Error())
				}
			}
		}()
	}

	// Keep this job running forever
	delayNextRestart := false
	for {

		// Is the context done?
		select {
		case <-ctx.Done():
			return
		default:
		}

		if delayNextRestart {
			jc.log.Infow("restarting job", "delay", jc.RestartDelay)
			if jc.RestartDelay > 0 {
				time.Sleep(jc.RestartDelay)
			}
		} else {
			jc.log.Info("running job")
			delayNextRestart = true
		}

		metrics.RecordInc(ctx, metrics.JobStart)
		err := jc.Job.Run(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			if errors.Is(err, context.DeadlineExceeded) {
				metrics.RecordInc(ctx, metrics.JobTimeout)
				delayNextRestart = false
			} else {
				metrics.RecordInc(ctx, metrics.JobError)
			}
			jc.log.Errorw("job exited with failure", "error", err.Error())
			jc.errorMsg = err.Error()

			if !jc.RestartOnFailure {
				// Exit the job
				break
			}
		} else {
			metrics.RecordInc(ctx, metrics.JobComplete)
			jc.log.Info("job exited cleanly")

			if !jc.RestartOnCompletion {
				// Exit the job
				break
			}
		}
	}
}

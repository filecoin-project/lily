package schedule_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/fxtest"

	"github.com/filecoin-project/lily/schedule"
)

func newTestJob() *testJob {
	return &testJob{
		errChan: make(chan error),
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

type testJob struct {
	// for causing the Run method to retrun an err
	errChan chan error
	// for blocking until the job is running
	started chan struct{}
	// for blocking until the job is stopped
	stopped chan struct{}
}

func (r *testJob) Run(ctx context.Context) error {
	r.started <- struct{}{}
	defer func() {
		r.stopped <- struct{}{}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chanErr := <-r.errChan:
			return chanErr
		}
	}
}

func TestScheduler(t *testing.T) {
	t.Run("Scheduler List Jobs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		tJob := newTestJob()

		s := schedule.NewScheduler(0, &schedule.JobConfig{
			Name:                t.Name(),
			Job:                 tJob,
			RestartOnFailure:    false,
			RestartOnCompletion: false,
			RestartDelay:        0,
		})

		go func() {
			err := s.Run(ctx)
			assert.Equal(t, context.Canceled, err)
		}()

		// wait for it to start
		<-tJob.started

		jobs := s.Jobs()
		assert.Len(t, jobs, 1)
		assert.True(t, jobs[0].Running)
		assert.Equal(t, schedule.JobID(1), jobs[0].ID)
		assert.Equal(t, jobs[0].Name, t.Name())
	})

	t.Run("Scheduler Daemon Submit and List Jobs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		// should be no jobs on start
		jobs := s.Jobs()
		assert.Len(t, jobs, 0)

		tJob := newTestJob()
		jobID := s.Submit(&schedule.JobConfig{
			Name:                t.Name(),
			Job:                 tJob,
			RestartOnFailure:    false,
			RestartOnCompletion: false,
			RestartDelay:        0,
		})

		// wait for it to start
		<-tJob.started

		jobs = s.Jobs()
		assert.Len(t, jobs, 1)
		assert.Equal(t, jobs[0].ID, jobID)
		assert.True(t, jobs[0].Running, true)
		assert.Equal(t, jobs[0].Name, t.Name())
	})

	t.Run("Scheduler Daemon start and stop job", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		// Stopping a job that Dne should fail with error
		assert.Error(t, s.StopJob(schedule.InvalidJobID))
		// Starting a job that Dne should fail with error
		assert.Error(t, s.StartJob(schedule.InvalidJobID))

		tJob := newTestJob()
		jobID := s.Submit(&schedule.JobConfig{
			Name:                t.Name(),
			Job:                 tJob,
			RestartOnFailure:    false,
			RestartOnCompletion: false,
			RestartDelay:        0,
		})
		// wait for job to start an assert it started correctly
		<-tJob.started
		jobs := s.Jobs()
		assert.Len(t, jobs, 1)
		assert.Equal(t, jobs[0].ID, jobID)
		assert.True(t, jobs[0].Running)
		assert.Equal(t, jobs[0].Name, t.Name())

		// wait for the job to stop and assert it is no longer running
		assert.NoError(t, s.StopJob(jobID))
		<-tJob.stopped

		jobs = s.Jobs()
		assert.Len(t, jobs, 1)
		assert.Equal(t, jobs[0].ID, jobID)
		assert.False(t, jobs[0].Running)
		assert.Equal(t, jobs[0].Name, t.Name())

		// stopping a job that is already stopped should fail
		assert.Error(t, s.StopJob(jobID))

		// ensure the job can be started again
		assert.NoError(t, s.StartJob(jobID))
		<-tJob.started

		jobs = s.Jobs()
		assert.Len(t, jobs, 1)
		assert.Equal(t, jobs[0].ID, jobID)
		assert.True(t, jobs[0].Running)
		assert.Equal(t, jobs[0].Name, t.Name())

		// starting a job twice should error
		assert.Error(t, s.StartJob(jobID))
	})

	t.Run("Job restarts on failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		tJob := newTestJob()
		_ = s.Submit(&schedule.JobConfig{
			Name:                t.Name(),
			Job:                 tJob,
			RestartOnFailure:    true,
			RestartOnCompletion: false,
			RestartDelay:        0,
		})
		// wait for job to start
		<-tJob.started

		// ensure the job is running
		jobs := s.Jobs()
		assert.True(t, jobs[0].Running)

		// cause the job to return an error
		tJob.errChan <- errors.New("FAIL")

		// the job should remain running
		jobs = s.Jobs()
		assert.True(t, jobs[0].Running)
	})
}

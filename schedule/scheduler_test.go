package schedule_test

import (
	"context"
	"errors"
	"testing"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/fxtest"

	"github.com/filecoin-project/lily/schedule"
)

func newTestJob(fn func(ctx context.Context) error) *testJob {
	return &testJob{
		fn: fn,
	}
}

type testJob struct {
	// for injecting a job
	fn   func(ctx context.Context) error
	done chan struct{}
}

func (r *testJob) Run(ctx context.Context) error {
	r.done = make(chan struct{})
	defer close(r.done)
	return r.fn(ctx)
}

func (r *testJob) Done() <-chan struct{} {
	return r.done
}

func TestScheduler(t *testing.T) {
	t.Run("Job Submit", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		done := make(chan struct{})
		_ = s.Submit(&schedule.JobConfig{
			Job: newTestJob(func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				close(done)
				return nil
			}),
			Name: t.Name(),
		})
		select {
		case <-done:
			t.Fatal("Submit did not return immediately")
		default:
		}
	})

	t.Run("Job Submit and wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		stop := make(chan struct{})
		job := newTestJob(func(ctx context.Context) error {
			<-stop
			return nil
		})
		s.Submit(&schedule.JobConfig{
			Job:  job,
			Name: t.Name(),
		})
		// wait for job to execute
		time.Sleep(100 * time.Millisecond)
		// ensure the job is running
		jobs := s.Jobs()
		assert.True(t, jobs[0].Running)

		// stop the job
		close(stop)
		select {
		case <-time.Tick(time.Millisecond * 500):
			t.Fatal("job is not done")
		case <-job.Done():
		}
		// wait for scheduler to clean up
		time.Sleep(100 * time.Millisecond)
		// job is not running
		jobs = s.Jobs()
		assert.False(t, jobs[0].Running)

	})

	t.Run("Job start and stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		s.Submit(&schedule.JobConfig{
			Job: newTestJob(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}),
		})
		for i := 0; i < 10; i++ {
			// ensure the job is running
			time.Sleep(100 * time.Millisecond)
			jobs := s.Jobs()
			assert.True(t, jobs[0].Running)

			// starting a running job errors
			assert.Error(t, s.StartJob(1))

			// ensure the job is stopped
			assert.NoError(t, s.StopJob(1))
			time.Sleep(100 * time.Millisecond)
			jobs = s.Jobs()
			assert.False(t, jobs[0].Running)

			// stopping a stopped job errors
			assert.Error(t, s.StopJob(1))

			// ensure the job is running
			assert.NoError(t, s.StartJob(1))
			time.Sleep(100 * time.Millisecond)
			jobs = s.Jobs()
			assert.True(t, jobs[0].Running)
		}
	})

	t.Run("Job start, list:running=true, stop, list:running=false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		stop := make(chan struct{})
		s.Submit(&schedule.JobConfig{
			Job: newTestJob(func(ctx context.Context) error {
				<-stop
				return nil
			}),
			Name: t.Name(),
		})
		// wait for job to execute
		time.Sleep(100 * time.Millisecond)
		// ensure the job is running
		jobs := s.Jobs()
		assert.True(t, jobs[0].Running)

		// stop the job
		close(stop)
		// wait for scheduler to clean up
		time.Sleep(100 * time.Millisecond)
		// job is not running
		jobs = s.Jobs()
		assert.False(t, jobs[0].Running)

	})

	t.Run("Job restart on success success, list:running=true, stop, list:running=true", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		stop := make(chan struct{})
		s.Submit(&schedule.JobConfig{
			Job: newTestJob(func(ctx context.Context) error {
				<-stop
				return nil
			}),
			Name:                t.Name(),
			RestartOnCompletion: true,
		})
		// wait for job to execute
		time.Sleep(100 * time.Millisecond)
		// ensure the job is running
		jobs := s.Jobs()
		assert.True(t, jobs[0].Running)

		// stop the job
		close(stop)
		// wait for scheduler to clean up
		time.Sleep(100 * time.Millisecond)
		// job is running
		jobs = s.Jobs()
		assert.True(t, jobs[0].Running)

	})

	t.Run("Job restart on error success, list:running=true, stop, list:running=true", func(t *testing.T) {
		// scheduler logs are noisy when job returns an error
		logging.SetAllLoggers(logging.LevelFatal)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := schedule.NewSchedulerDaemon(ctx, fxtest.NewLifecycle(t))

		stop := make(chan struct{})
		s.Submit(&schedule.JobConfig{
			Job: newTestJob(func(ctx context.Context) error {
				<-stop
				return errors.New("error")
			}),
			Name:             t.Name(),
			RestartOnFailure: true,
		})
		// wait for job to execute
		time.Sleep(100 * time.Millisecond)
		// ensure the job is running
		jobs := s.Jobs()
		assert.True(t, jobs[0].Running)

		// stop the job
		close(stop)
		// wait for scheduler to clean up
		time.Sleep(100 * time.Millisecond)
		// job is running
		jobs = s.Jobs()
		assert.True(t, jobs[0].Running)

	})
}

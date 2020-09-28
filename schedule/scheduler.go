package schedule

import (
	"context"
	"errors"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
)

var log = logging.Logger("schedule")

type Task interface {
	// Run starts running the task and blocks until the context is done or
	// an error occurs.
	Run(context.Context) error
}

type TaskConfig struct {
	// Name is a human readable name for the task for use in logging
	Name string

	// Task is the task that will be executed.
	Task Task

	// Locker is an optional lock that must be taken before the task can execute.
	Locker Locker

	// RestartOnFailure controls whether the task should be restarted if it fails with an error.
	RestartOnFailure bool

	// RestartOnCompletion controls whether the task should be restarted if it fails without an error.
	RestartOnCompletion bool
}

// Locker represents a general lock that a task may need to take before operating.
type Locker interface {
	Lock(context.Context) error
	Unlock(context.Context) error
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

type Scheduler struct {
	tasks []TaskConfig
}

// Add add a task config to the scheduler. This must not be called after Run.
func (s *Scheduler) Add(tc TaskConfig) error {
	s.tasks = append(s.tasks, tc)
	return nil
}

// Run starts running the scheduler and blocks until the context is done or
// all tasks have run to completion.
func (s *Scheduler) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskComplete := make(chan struct{}, len(s.tasks))
	tasksRunning := len(s.tasks)

	for _, tc := range s.tasks {
		go func(tc TaskConfig) {
			// Report task is complete when this goroutine exits
			defer func() {
				taskComplete <- struct{}{}
			}()

			// Attempt to get the task lock if specified
			if tc.Locker != nil {
				if err := tc.Locker.Lock(ctx); err != nil {
					if errors.Is(err, storage.ErrLockNotAcquired) {
						log.Infow("task not started: lock not acquired", "task", tc.Name)
						return
					}
					log.Errorw("task not started: lock not acquired", "task", tc.Name, "error", err.Error())
					return
				}
				defer func() {
					if err := tc.Locker.Unlock(ctx); err != nil {
						if !errors.Is(err, context.Canceled) {
							log.Errorw("failed to unlock task", "task", tc.Name, "error", err.Error())
						}
					}
				}()
			}

			// Keep this task running forever
			for {

				// Is the context done?
				select {
				case <-ctx.Done():
					return
				default:
				}

				log.Infow("running task", "task", tc.Name)
				err := tc.Task.Run(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						break
					}
					if !tc.RestartOnFailure {
						// Exit the task
						log.Errorw("task exited with failure, not restarting", "task", tc.Name, "error", err.Error())
						break
					}
					log.Errorw("task exited with failure, restarting", "task", tc.Name, "error", err.Error())
				} else {
					if !tc.RestartOnCompletion {
						// Exit the task
						log.Infow("task exited cleanly, not restarting", "task", tc.Name)
						break
					}
					log.Infow("task exited cleanly, restarting", "task", tc.Name)
				}
			}

		}(tc)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// A little jitter between tasks to reduce thundering herd effects on api
		wait.SleepWithJitter(500*time.Millisecond, 2)
	}

	// Wait until the context is done or all tasks have been completed
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-taskComplete:
			// A task has completed
			tasksRunning--
			if tasksRunning == 0 {
				// All tasks have completed successfully.
				return nil
			}
		}
	}

}

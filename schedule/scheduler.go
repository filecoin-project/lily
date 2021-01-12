package schedule

import (
	"context"
	"errors"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

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

	// RestartOnFailure controls whether the task should be restarted if it stops with an error.
	RestartOnFailure bool

	// RestartOnCompletion controls whether the task should be restarted if it stops without an error.
	RestartOnCompletion bool

	// RestartDelay is the amount of time to wait before restarting a stopped task
	RestartDelay time.Duration
}

// Locker represents a general lock that a task may need to take before operating.
type Locker interface {
	Lock(context.Context) error
	Unlock(context.Context) error
}

func NewScheduler(taskDelay time.Duration) *Scheduler {
	// Enforce a minimum delay
	if taskDelay == 0 {
		taskDelay = 100 * time.Millisecond
	}
	return &Scheduler{
		taskDelay: taskDelay,
	}
}

type Scheduler struct {
	tasks     []TaskConfig
	taskDelay time.Duration
}

// Add add a task config to the scheduler. This must not be called after Run.
func (s *Scheduler) Add(tc TaskConfig) {
	s.tasks = append(s.tasks, tc)
}

// Run starts running the scheduler and blocks until the context is done or
// all tasks have run to completion.
func (s *Scheduler) Run(ctx context.Context) error {
	if len(s.tasks) == 0 {
		return xerrors.Errorf("no tasks to run")
	}
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
			doneFirstRun := false
			for {

				// Is the context done?
				select {
				case <-ctx.Done():
					return
				default:
				}

				if doneFirstRun {
					log.Infow("restarting task", "task", tc.Name, "delay", tc.RestartDelay)
					if tc.RestartDelay > 0 {
						time.Sleep(tc.RestartDelay)
					}
				} else {
					log.Infow("running task", "task", tc.Name)
					doneFirstRun = true
				}

				err := tc.Task.Run(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						break
					}
					log.Errorw("task exited with failure", "task", tc.Name, "error", err.Error())

					if !tc.RestartOnFailure {
						// Exit the task
						break
					}
				} else {
					log.Infow("task exited cleanly", "task", tc.Name)

					if !tc.RestartOnCompletion {
						// Exit the task
						break
					}
				}
			}
		}(tc)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// A little jitter between tasks to reduce thundering herd effects on api
		wait.SleepWithJitter(s.taskDelay, 2)
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

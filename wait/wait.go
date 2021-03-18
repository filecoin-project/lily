package wait

import (
	"context"
	"math/rand"
	"time"
)

// A CheckFunc returns true when the check has been passed and false if it has not.
type CheckFunc func(context.Context) (bool, error)

// RepeatUntil runs c every period until the context is done, c returns an error or c returns true to indicate completion.
func RepeatUntil(ctx context.Context, period time.Duration, c CheckFunc) error {
	timer := time.NewTimer(period)

	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Perform the check
		done, err := c(ctx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		// Shortcut the timer if there is no wait period
		if period == 0 {
			continue
		}

		// Wait for the next check
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(period)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Until runs f until the context is done or f returns.
func Until(ctx context.Context, f func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errch := make(chan error, 1)

	go func() {
		errch <- f(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	}
}

// Jitter returns a random duration ranging from base to base+base*factor
func Jitter(base time.Duration, factor float64) time.Duration {
	//nolint:gosec
	return base + time.Duration(float64(base)*factor*rand.Float64())
}

// SleepWithJitter sleeps for a random duration ranging from base to base+base*factor
func SleepWithJitter(base time.Duration, factor float64) {
	time.Sleep(Jitter(base, factor))
}

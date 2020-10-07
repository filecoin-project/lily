package storage

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

var ErrLockNotAcquired = errors.New("lock not acquired")
var ErrLockNotReleased = errors.New("lock not released")

// An AdvisoryLock is a lock that is managed by Postgres but is only enforced by the application. Advisory
// locks are automatically released at the end of a session. It is safe to hold both a shared and exclusive
// lock within a single session.
type AdvisoryLock int64

// LockShared tries to acquire a session scoped exclusive advisory lock.
func (l AdvisoryLock) LockExclusive(ctx context.Context, db *pg.DB) error {
	var acquired bool
	_, err := db.QueryOneContext(ctx, pg.Scan(&acquired), `SELECT pg_try_advisory_lock(?);`, int64(l))
	if err != nil {
		return xerrors.Errorf("acquiring exclusive lock: %w", err)
	}
	if !acquired {
		return ErrLockNotAcquired
	}
	return nil
}

// UnlockExclusive releases an exclusive advisory lock.
func (l AdvisoryLock) UnlockExclusive(ctx context.Context, db *pg.DB) error {
	var released bool
	_, err := db.QueryOneContext(ctx, pg.Scan(&released), `SELECT pg_advisory_unlock(?);`, int64(l))
	if err != nil {
		return xerrors.Errorf("unlocking exclusive lock: %w", err)
	}
	if !released {
		return ErrLockNotReleased
	}
	return nil
}

// LockShared tries to acquire a session scoped shared advisory lock.
func (l AdvisoryLock) LockShared(ctx context.Context, db *pg.DB) error {
	var acquired bool
	_, err := db.QueryOneContext(ctx, pg.Scan(&acquired), `SELECT pg_try_advisory_lock_shared(?);`, int64(l))
	if err != nil {
		return xerrors.Errorf("acquiring exclusive lock: %w", err)
	}
	if !acquired {
		return ErrLockNotAcquired
	}
	return nil
}

// UnlockShared releases a shared advisory lock.
func (l AdvisoryLock) UnlockShared(ctx context.Context, db *pg.DB) error {
	var released bool
	_, err := db.QueryOneContext(ctx, pg.Scan(&released), `SELECT pg_advisory_unlock_shared(?);`, int64(l))
	if err != nil {
		return xerrors.Errorf("unlocking shared lock: %w", err)
	}
	if !released {
		return ErrLockNotReleased
	}
	return nil
}

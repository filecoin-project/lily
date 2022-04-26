package watch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/cache"
	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/metrics"
)

var log = logging.Logger("lily/chain/watch")

type WatcherAPI interface {
	Observe(obs events.TipSetObserver) *types.TipSet
	//Unregister(obs events.TipSetObserver) bool
	ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)
}

type WatcherOpt func(w *Watcher)

func WithTasks(tasks ...string) WatcherOpt {
	return func(w *Watcher) {
		w.tasks = tasks
	}
}

func WithConfidence(c int) WatcherOpt {
	return func(w *Watcher) {
		w.confidence = c
	}
}

func WithConcurrentWorkers(p int) WatcherOpt {
	return func(w *Watcher) {
		w.poolSize = p
	}
}

func WithBufferSize(b int) WatcherOpt {
	return func(w *Watcher) {
		w.bufferSize = b
	}
}

// Watcher is a task that indexes blocks by following the chain head.
type Watcher struct {
	// required
	api  WatcherAPI
	name string

	// options with defaults
	confidence int // size of tipset cache
	bufferSize int // size of the buffer for incoming tipset notifications.
	poolSize   int
	tasks      []string

	// created internally
	done       chan struct{}
	indexer    indexer.Indexer
	cache      *cache.TipSetCache     // caches tipsets for possible reversion
	pool       *workerpool.WorkerPool // used for async tipset indexing
	tsObserver *TipSetObserver

	// metric tracking
	active int64 // must be accessed using atomic operations, updated automatically.

	// error handling
	fatalMu sync.Mutex
	fatal   error
}

var (
	WatcherDefaultBufferSize        = 5
	WatcherDefaultConfidence        = 1
	WatcherDefaultConcurrentWorkers = 1
	WatcherDefaultTasks             = tasktype.AllTableTasks
)

// NewWatcher creates a new Watcher. confidence sets the number of tipsets that will be held
// in a cache awaiting possible reversion. Tipsets will be written to the database when they are evicted from
// the cache due to incoming later tipsets.
func NewWatcher(api WatcherAPI, indexer indexer.Indexer, name string, opts ...WatcherOpt) *Watcher {
	w := &Watcher{
		api:     api,
		name:    name,
		indexer: indexer,

		bufferSize: WatcherDefaultBufferSize,
		confidence: WatcherDefaultConfidence,
		poolSize:   WatcherDefaultConcurrentWorkers,
		tasks:      WatcherDefaultTasks,
	}

	for _, opt := range opts {
		opt(w)
	}
	return w
}

func (c *Watcher) init(ctx context.Context) error {
	c.done = make(chan struct{})
	c.pool = workerpool.New(c.poolSize)

	c.tsObserver = &TipSetObserver{bufferSize: c.bufferSize}
	head := c.api.Observe(c.tsObserver)
	if err := c.tsObserver.SetCurrent(ctx, head); err != nil {
		return err
	}

	c.cache = cache.NewTipSetCache(c.confidence)
	if err := c.cache.Warm(ctx, head, c.api.ChainGetTipSet); err != nil {
		return err
	}

	return nil
}

func (c *Watcher) close() {
	// ensure we clear the fatal error after shut down, this allows the watcher to be restarted without reinitializing its state.
	c.setFatalError(nil)
	// ensure we shut down the pool when the watcher stops.
	c.pool.Stop()
	// ensure we reset the tipset cache to avoid process stale state if watcher is restarted.
	c.cache.Reset()
	// unregister the observer
	// TODO https://github.com/filecoin-project/lotus/pull/8441
	//c.api.Unregister(notifier)
	// close channel to signal completion
	close(c.done)
}

// Run starts following the chain head and blocks until the context is done or
// an error occurs.
func (c *Watcher) Run(ctx context.Context) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	defer c.close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case he, ok := <-c.tsObserver.HeadEvents():
			if !ok {
				return c.tsObserver.Err()
			}
			if he != nil && he.TipSet != nil {
				metrics.RecordCount(ctx, metrics.WatchHeight, int(he.TipSet.Height()))
			}

			if err := c.index(ctx, he); err != nil {
				return xerrors.Errorf("index: %w", err)
			}
		}
	}

}

func (c *Watcher) Done() <-chan struct{} {
	return c.done
}

func (c *Watcher) index(ctx context.Context, he *HeadEvent) error {
	switch he.Type {
	case HeadEventCurrent:
		err := c.cache.SetCurrent(he.TipSet)
		if err != nil {
			log.Errorw("tipset cache set current", "error", err.Error(), "reporter", c.name)
		}

		// If we have a zero confidence window then we need to notify every tipset we see
		if c.confidence == 0 {
			if err := c.indexTipSetAsync(ctx, he.TipSet); err != nil {
				return xerrors.Errorf("notify tipset: %w", err)
			}
		}
	case HeadEventApply:
		tail, err := c.cache.Add(he.TipSet)
		if err != nil {
			log.Errorw("tipset cache add", "error", err.Error(), "reporter", c.name)
		}

		// Send the tipset that fell out of the confidence window to the observer
		if tail != nil {
			if err := c.indexTipSetAsync(ctx, tail); err != nil {
				return xerrors.Errorf("notify tipset: %w", err)
			}
		}

	case HeadEventRevert:
		err := c.cache.Revert(he.TipSet)
		if err != nil {
			if errors.Is(err, cache.ErrEmptyRevert) {
				// The chain is unwinding but our cache is empty. This probably means we have already processed
				// the tipset being reverted and may process it again or an alternate heaviest tipset for this height.
				metrics.RecordInc(ctx, metrics.TipSetCacheEmptyRevert)
			}
			log.Errorw("tipset cache revert", "error", err.Error(), "reporter", c.name)
		}
	}

	metrics.RecordCount(ctx, metrics.TipSetCacheSize, c.cache.Size())
	metrics.RecordCount(ctx, metrics.TipSetCacheDepth, c.cache.Len())

	log.Debugw("tipset cache", "height", c.cache.Height(), "tail_height", c.cache.TailHeight(), "length", c.cache.Len(), "reporter", c.name)

	return nil
}

// indexTipSetAsync is called when a new tipset has been discovered
func (c *Watcher) indexTipSetAsync(ctx context.Context, ts *types.TipSet) error {
	if err := c.fatalError(); err != nil {
		return err
	}

	stats.Record(ctx, metrics.WatcherActiveWorkers.M(c.active))
	stats.Record(ctx, metrics.WatcherWaitingWorkers.M(int64(c.pool.WaitingQueueSize())))
	if c.pool.WaitingQueueSize() > c.pool.Size() {
		log.Warnw("queuing worker in watcher pool", "waiting", c.pool.WaitingQueueSize(), "reporter", c.name)
	}
	log.Infow("submitting tipset for async indexing", "height", ts.Height(), "active", c.active, "reporter", c.name)

	ctx, span := otel.Tracer("").Start(ctx, "Watcher.indexTipSetAsync")
	c.pool.Submit(func() {
		atomic.AddInt64(&c.active, 1)
		defer func() {
			atomic.AddInt64(&c.active, -1)
			span.End()
		}()

		ts := ts
		success, err := c.indexer.TipSet(ctx, ts, indexer.WithIndexerType(indexer.Watch), indexer.WithTasks(c.tasks))
		if err != nil {
			log.Errorw("watcher suffered fatal error", "error", err, "height", ts.Height(), "tipset", ts.Key().String(), "reporter", c.name)
			c.setFatalError(err)
			return
		}
		if !success {
			log.Warnw("watcher failed to fully index tipset", "height", ts.Height(), "tipset", ts.Key().String(), "reporter", c.name)
		}
	})
	return nil
}

func (c *Watcher) setFatalError(err error) {
	c.fatalMu.Lock()
	c.fatal = err
	c.fatalMu.Unlock()
}

func (c *Watcher) fatalError() error {
	c.fatalMu.Lock()
	out := c.fatal
	c.fatalMu.Unlock()
	return out
}

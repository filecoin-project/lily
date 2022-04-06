package chain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	"go.opencensus.io/stats"
	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/metrics"
)

type WatcherAPI interface {
	Observe(obs events.TipSetObserver) *types.TipSet
	//Unregister(obs events.TipSetObserver) bool
	ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)
}

// NewWatcher creates a new Watcher. confidence sets the number of tipsets that will be held
// in a cache awaiting possible reversion. Tipsets will be written to the database when they are evicted from
// the cache due to incoming later tipsets.
func NewWatcher(api WatcherAPI, indexer *indexer.Manager, confidence int, poolSize int, bufferSize int) *Watcher {
	return &Watcher{
		api:        api,
		bufferSize: bufferSize,
		indexer:    indexer,
		confidence: confidence,
		cache:      NewTipSetCache(confidence),
		poolSize:   poolSize,
	}
}

// Watcher is a task that indexes blocks by following the chain head.
type Watcher struct {
	api        WatcherAPI
	indexer    *indexer.Manager
	confidence int          // size of tipset cache
	bufferSize int          // size of the buffer for incoming tipset notifications.
	cache      *TipSetCache // caches tipsets for possible reversion
	done       chan struct{}

	// used for async tipset indexing
	poolSize int
	pool     *workerpool.WorkerPool
	active   int64 // must be accessed using atomic operations, updated automatically.

	fatalMu sync.Mutex
	fatal   error
}

// Run starts following the chain head and blocks until the context is done or
// an error occurs.
func (c *Watcher) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	c.done = make(chan struct{})

	notifier := &TipSetObserver{bufferSize: c.bufferSize}
	head := c.api.Observe(notifier)
	if err := notifier.SetCurrent(ctx, head); err != nil {
		return err
	}
	if err := c.cache.Warm(ctx, head, c.api.ChainGetTipSet); err != nil {
		return err
	}
	c.pool = workerpool.New(c.poolSize)

	defer func() {
		// ensure we shut down the pool when the watcher stops.
		c.pool.Stop()
		// ensure we clear the fatal error after shut down, this allows the watcher to be restarted without reinitializing its state.
		c.setFatalError(nil)
		// ensure we reset the tipset cache to avoid process stale state if watcher is restarted.
		c.cache.Reset()
		// unregister the observer
		// TODO https://github.com/filecoin-project/lotus/pull/8441
		//c.api.Unregister(notifier)
		close(c.done)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case he, ok := <-notifier.HeadEvents():
			if !ok {
				return notifier.Err()
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
			log.Errorw("tipset cache set current", "error", err.Error())
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
			log.Errorw("tipset cache add", "error", err.Error())
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
			if errors.Is(err, ErrEmptyRevert) {
				// The chain is unwinding but our cache is empty. This probably means we have already processed
				// the tipset being reverted and may process it again or an alternate heaviest tipset for this height.
				metrics.RecordInc(ctx, metrics.TipSetCacheEmptyRevert)
			}
			log.Errorw("tipset cache revert", "error", err.Error())
		}
	}

	metrics.RecordCount(ctx, metrics.TipSetCacheSize, c.cache.Size())
	metrics.RecordCount(ctx, metrics.TipSetCacheDepth, c.cache.Len())

	log.Debugw("tipset cache", "height", c.cache.Height(), "tail_height", c.cache.TailHeight(), "length", c.cache.Len())

	return nil
}

// indexTipSetAsync is called when a new tipset has been discovered
func (c *Watcher) indexTipSetAsync(ctx context.Context, ts *types.TipSet) error {
	if err := c.fatalError(); err != nil {
		defer c.pool.Stop()
		return err
	}

	stats.Record(ctx, metrics.WatcherActiveWorkers.M(c.active))
	stats.Record(ctx, metrics.WatcherWaitingWorkers.M(int64(c.pool.WaitingQueueSize())))
	if c.pool.WaitingQueueSize() > c.pool.Size() {
		log.Warnw("queuing worker in watcher pool", "waiting", c.pool.WaitingQueueSize())
	}
	log.Infow("submitting tipset for async indexing", "height", ts.Height(), "active", c.active)

	ctx, span := otel.Tracer("").Start(ctx, "Manager.TipSetAsync")
	c.pool.Submit(func() {
		atomic.AddInt64(&c.active, 1)
		defer func() {
			atomic.AddInt64(&c.active, -1)
			span.End()
		}()

		ts := ts
		success, err := c.indexer.TipSet(ctx, ts)
		if err != nil {
			log.Errorw("watcher suffered fatal error", "error", err, "height", ts.Height(), "tipset", ts.Key().String())
			c.setFatalError(err)
			return
		}
		if !success {
			log.Warnw("watcher failed to fully index tipset", "height", ts.Height(), "tipset", ts.Key().String())
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

// A HeadNotifier reports tipset events that occur at the head of the chain
type HeadNotifier interface {
	// HeadEvents returns a channel that receives head events. It may be closed
	// by the sender of the events, in which case Err will return a non-nil error
	// explaining why. HeadEvents may return nil if this implementation will never
	// notify any events.
	HeadEvents() <-chan *HeadEvent

	// Err returns the reason for the closing of the HeadEvents channel.
	Err() error
}

// A HeadEvent is a notification of a change at the head of the chain
type HeadEvent struct {
	Type   string
	TipSet *types.TipSet
}

// Constants for HeadEvent types
const (
	// HeadEventRevert indicates that the event signals a reversion of a tipset from the chain
	HeadEventRevert = "revert"

	// HeadEventRevert indicates that the event signals the application of a tipset to the chain
	HeadEventApply = "apply"

	// HeadEventRevert indicates that the event signals the current known head tipset
	HeadEventCurrent = "current"
)

var _ events.TipSetObserver = (*TipSetObserver)(nil)

type TipSetObserver struct {
	mu     sync.Mutex      // protects following fields
	events chan *HeadEvent // created lazily, closed by first cancel call
	err    error           // set to non-nil by the first cancel call

	// size of the buffer to maintain for events. Using a buffer reduces chance
	// that the emitter of events will block when sending to this notifier.
	bufferSize int
}

func (h *TipSetObserver) eventsCh() chan *HeadEvent {
	// caller must hold mu
	if h.events == nil {
		h.events = make(chan *HeadEvent, h.bufferSize)
	}
	return h.events
}

func (h *TipSetObserver) HeadEvents() <-chan *HeadEvent {
	h.mu.Lock()
	ev := h.eventsCh()
	h.mu.Unlock()
	return ev
}

func (h *TipSetObserver) Err() error {
	h.mu.Lock()
	err := h.err
	h.mu.Unlock()
	return err
}

func (h *TipSetObserver) Cancel(err error) {
	h.mu.Lock()
	if h.err != nil {
		h.mu.Unlock()
		return
	}
	h.err = err
	if h.events == nil {
		h.events = make(chan *HeadEvent, h.bufferSize)
	}
	close(h.events)
	h.mu.Unlock()
}

func (h *TipSetObserver) SetCurrent(ctx context.Context, ts *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	// This is imprecise since it's inherently racy but good enough to emit
	// a warning that the event may block the sender
	if len(ev) == cap(ev) {
		log.Warnw("head notifier buffer at capacity", "queued", len(ev))
	}

	log.Debugw("head notifier setting head", "tipset", ts.Key().String())
	ev <- &HeadEvent{
		Type:   HeadEventCurrent,
		TipSet: ts,
	}
	return nil
}

func (h *TipSetObserver) Apply(ctx context.Context, from, to *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	// This is imprecise since it's inherently racy but good enough to emit
	// a warning that the event may block the sender
	if len(ev) == cap(ev) {
		log.Warnw("head notifier buffer at capacity", "queued", len(ev))
	}

	log.Debugw("head notifier apply", "to", to.Key().String(), "from", from.Key().String())
	select {
	case ev <- &HeadEvent{
		Type:   HeadEventApply,
		TipSet: to,
	}:
	default:
		log.Errorw("head notifier event channel blocked dropping apply event", "to", to.Key().String(), "from", from.Key().String())
	}
	return nil
}

func (h *TipSetObserver) Revert(ctx context.Context, from, to *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	// This is imprecise since it's inherently racy but good enough to emit
	// a warning that the event may block the sender
	if len(ev) == cap(ev) {
		log.Warnw("head notifier buffer at capacity", "queued", len(ev))
	}

	log.Debugw("head notifier revert", "to", to.Key().String(), "from", from.Key().String())
	select {
	case ev <- &HeadEvent{
		Type:   HeadEventRevert,
		TipSet: from,
	}:
	default:
		log.Errorw("head notifier event channel blocked dropping revert event", "to", to.Key().String(), "from", from.Key().String())
	}
	return nil
}

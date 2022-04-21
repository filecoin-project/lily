package watch

import (
	"context"
	"sync"

	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/types"
)

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

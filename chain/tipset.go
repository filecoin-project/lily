package chain

import (
	"context"
	"errors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
)

// A TipSetObserver waits for notifications of new tipsets.
type TipSetObserver interface {
	TipSet(ctx context.Context, ts *types.TipSet) error
	SkipTipSet(ctx context.Context, ts *types.TipSet, reason string) error
	Close() error
}

var (
	ErrCacheEmpty       = errors.New("cache empty")
	ErrAddOutOfOrder    = errors.New("added tipset height lower than current head")
	ErrRevertOutOfOrder = errors.New("reverted tipset does not match current head")
)

// TipSetCache is a cache of recent tipsets that can keep track of reversions.
// Inspired by tipSetCache in Lotus chain/events package.
type TipSetCache struct {
	buffer  []*types.TipSet
	idxHead int // idxHead is the current position of the head tipset in the buffer
	len     int // len is the number of items in the cache
}

func NewTipSetCache(size int) *TipSetCache {
	return &TipSetCache{
		buffer: make([]*types.TipSet, size),
	}
}

// Head returns the tipset at the head of the cache.
func (c *TipSetCache) Head() (*types.TipSet, error) {
	if c.len == 0 {
		return nil, ErrCacheEmpty
	}
	return c.buffer[c.idxHead], nil
}

// Tail returns the tipset at the tail of the cache.
func (c *TipSetCache) Tail() (*types.TipSet, error) {
	if c.len == 0 {
		return nil, ErrCacheEmpty
	}
	return c.buffer[c.tailIndex()], nil
}

// Add adds a new tipset which becomes the new head of the cache. If the buffer is full, the tail
// being evicted is also returned.
func (c *TipSetCache) Add(ts *types.TipSet) (*types.TipSet, error) {
	if c.len == 0 {
		// Special case for zero length caches, simply pass back the added tipset
		if len(c.buffer) == 0 {
			return ts, nil
		}

		c.buffer[c.idxHead] = ts
		c.len++
		return nil, nil
	}

	headHeight := c.buffer[c.idxHead].Height()
	if headHeight >= ts.Height() {
		return nil, ErrAddOutOfOrder
	}

	c.idxHead = normalModulo(c.idxHead+1, len(c.buffer))
	old := c.buffer[c.idxHead]
	c.buffer[c.idxHead] = ts
	if c.len < len(c.buffer) {
		c.len++
		return nil, nil
	}

	// Return old tipset that was displaced
	return old, nil
}

// Revert removes the head tipset
func (c *TipSetCache) Revert(ts *types.TipSet) error {
	if c.len == 0 {
		return nil
	}

	// Can only revert the most recent tipset
	if c.buffer[c.idxHead].Key() != ts.Key() {
		return ErrRevertOutOfOrder
	}

	c.buffer[c.idxHead] = nil
	c.idxHead = normalModulo(c.idxHead-1, len(c.buffer))
	c.len--

	return nil
}

// SetCurrent replaces the current head
func (c *TipSetCache) SetCurrent(ts *types.TipSet) error {
	for c.len > 0 && c.buffer[c.idxHead].Height() > ts.Height() {
		c.buffer[c.idxHead] = nil
		c.idxHead = normalModulo(c.idxHead-1, len(c.buffer))
		c.len--
	}

	if c.len == 0 {
		_, err := c.Add(ts)
		return err
	}

	c.buffer[c.idxHead] = ts
	return nil
}

// Len returns the number of tipsets in the cache. This will never exceed the size of the cache.
func (c *TipSetCache) Len() int {
	return c.len
}

// Height returns the height of the current head or zero if the cache is empty.
func (c *TipSetCache) Height() abi.ChainEpoch {
	if c.len == 0 {
		return 0
	}
	return c.buffer[c.idxHead].Height()
}

// TailHeight returns the height of the current tail or zero if the cache is empty.
func (c *TipSetCache) TailHeight() abi.ChainEpoch {
	if c.len == 0 {
		return 0
	}
	return c.buffer[c.tailIndex()].Height()
}

func (c *TipSetCache) tailIndex() int {
	return normalModulo(c.idxHead-c.len+1, len(c.buffer))
}

// Reset removes all tipsets from the cache
func (c *TipSetCache) Reset() {
	for i := range c.buffer {
		c.buffer[i] = nil
	}
	c.idxHead = 0
	c.len = 0
}

func normalModulo(n, m int) int {
	return ((n % m) + m) % m
}

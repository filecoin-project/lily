package cache

import (
	"context"
	"errors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"
)

var (
	ErrCacheEmpty       = errors.New("cache empty")
	ErrAddOutOfOrder    = errors.New("added tipset height lower than current head")
	ErrRevertOutOfOrder = errors.New("reverted tipset does not match current head")
	ErrEmptyRevert      = errors.New("reverted received on empty cache")
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

// Confidence returns the number of tipset that the cache must hold before tipsets are evicted on Add.
func (c *TipSetCache) Confidence() int {
	return len(c.buffer)
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
		return ErrEmptyRevert
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

// Size returns the maximum number of tipsets that may be present in the cache.
func (c *TipSetCache) Size() int {
	return len(c.buffer)
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

// Warm fills the TipSetCache with confidence tipsets so that subsequent calls to Add return a tipset.
func (c *TipSetCache) Warm(ctx context.Context, head *types.TipSet, getTipSetFn func(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)) error {
	cur := head
	tss := make([]*types.TipSet, 0, c.Confidence())
	for i := 0; i < c.Confidence(); i++ {
		if cur.Height() == 0 {
			break
		}
		var err error
		cur, err = getTipSetFn(ctx, cur.Parents())
		if err != nil {
			return err
		}
		tss = append(tss, cur)
	}
	for i := len(tss) - 1; i >= 0; i-- {
		expectNil, err := c.Add(tss[i])
		if err != nil {
			return err
		}
		if expectNil != nil {
			return xerrors.Errorf("unexpected tipset returned while warming tipset cache: %s", expectNil)
		}
	}
	return nil
}

func normalModulo(n, m int) int {
	return ((n % m) + m) % m
}

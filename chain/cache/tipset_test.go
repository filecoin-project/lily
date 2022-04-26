package cache

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	dummyCid cid.Cid
	dummyTs  *types.TipSet
)

func init() {
	dummyCid, _ = cid.Parse("bafkqaaa")
	dummyTs = mustMakeTs(nil, 1, dummyCid)
}

func TestTipSetCacheInvariants(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewTipSetCache(3)
		ts, err := c.Head()
		assert.Error(t, err, ErrCacheEmpty)
		assert.Nil(t, ts)

		ts, err = c.Tail()
		assert.Error(t, err, ErrCacheEmpty)
		assert.Nil(t, ts)

		err = c.Revert(dummyTs)
		assert.Error(t, err, ErrEmptyRevert)

		oldTail, err := c.Add(dummyTs)
		assert.NoError(t, err)
		assert.Nil(t, oldTail)

		assert.Equal(t, 1, c.Len())
	})

	t.Run("reset empties cache", func(t *testing.T) {
		c := NewTipSetCache(3)

		_, err := c.Add(dummyTs)
		assert.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		c.Reset()
		assert.Equal(t, 0, c.Len())

		ts, err := c.Head()
		assert.Error(t, err, ErrCacheEmpty)
		assert.Nil(t, ts)

		ts, err = c.Tail()
		assert.Error(t, err, ErrCacheEmpty)
		assert.Nil(t, ts)

		err = c.Revert(dummyTs)
		assert.Error(t, err, ErrEmptyRevert)

		_, err = c.Add(dummyTs)
		assert.NoError(t, err)
	})

	t.Run("head returns last added", func(t *testing.T) {
		c := NewTipSetCache(3)
		ts1 := mustMakeTs(nil, 1, dummyCid)
		_, err := c.Add(ts1)
		require.NoError(t, err)

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts1, head)

		ts2 := mustMakeTs(nil, 2, dummyCid)
		_, err = c.Add(ts2)
		require.NoError(t, err)

		head, err = c.Head()
		require.NoError(t, err)
		assert.Same(t, ts2, head)
	})

	t.Run("tail returns first added", func(t *testing.T) {
		c := NewTipSetCache(3)
		ts1 := mustMakeTs(nil, 1, dummyCid)
		_, err := c.Add(ts1)
		require.NoError(t, err)

		tail, err := c.Tail()
		require.NoError(t, err)
		assert.Same(t, ts1, tail)

		ts2 := mustMakeTs(nil, 2, dummyCid)
		_, err = c.Add(ts2)
		require.NoError(t, err)

		tail, err = c.Tail()
		require.NoError(t, err)
		assert.Same(t, ts1, tail)
	})

	t.Run("length is capped by size", func(t *testing.T) {
		c := NewTipSetCache(3)
		ts1 := mustMakeTs(nil, 1, dummyCid)
		_, err := c.Add(ts1)
		assert.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		ts2 := mustMakeTs(nil, 2, dummyCid)
		_, err = c.Add(ts2)
		assert.NoError(t, err)
		assert.Equal(t, 2, c.Len())

		ts3 := mustMakeTs(nil, 3, dummyCid)
		_, err = c.Add(ts3)
		assert.NoError(t, err)
		assert.Equal(t, 3, c.Len())

		ts4 := mustMakeTs(nil, 4, dummyCid)
		_, err = c.Add(ts4)
		assert.NoError(t, err)
		assert.Equal(t, 3, c.Len())
	})

	t.Run("buffer is a ring", func(t *testing.T) {
		c := NewTipSetCache(3)

		// Add first
		ts1 := mustMakeTs(nil, 1, dummyCid)
		oldTail1, err := c.Add(ts1)
		require.NoError(t, err)
		assert.Nil(t, oldTail1)

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts1, head)

		tail, err := c.Tail()
		require.NoError(t, err)
		assert.Same(t, ts1, tail)

		// Add second
		ts2 := mustMakeTs(nil, 2, dummyCid)
		oldTail2, err := c.Add(ts2)
		require.NoError(t, err)
		assert.Nil(t, oldTail2)

		head, err = c.Head()
		require.NoError(t, err)
		assert.Same(t, ts2, head)

		tail, err = c.Tail()
		require.NoError(t, err)
		assert.Same(t, ts1, tail)

		// Add third (buffer is at capacity)
		ts3 := mustMakeTs(nil, 3, dummyCid)
		oldTail3, err := c.Add(ts3)
		require.NoError(t, err)
		assert.Nil(t, oldTail3)

		head, err = c.Head()
		require.NoError(t, err)
		assert.Same(t, ts3, head)

		tail, err = c.Tail()
		require.NoError(t, err)
		assert.Same(t, ts1, tail)

		// Add fourth (bumps ts1 out of the buffer which is returned)
		ts4 := mustMakeTs(nil, 4, dummyCid)
		oldTail4, err := c.Add(ts4)
		require.NoError(t, err)
		assert.Same(t, ts1, oldTail4)

		head, err = c.Head()
		require.NoError(t, err)
		assert.Same(t, ts4, head)

		tail, err = c.Tail()
		require.NoError(t, err)
		assert.Same(t, ts2, tail)
	})

	t.Run("revert moves head back", func(t *testing.T) {
		c := NewTipSetCache(3)
		ts1 := mustMakeTs(nil, 1, dummyCid)
		_, err := c.Add(ts1)
		require.NoError(t, err)

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts1, head)

		ts2 := mustMakeTs(nil, 2, dummyCid)
		_, err = c.Add(ts2)
		require.NoError(t, err)

		head, err = c.Head()
		require.NoError(t, err)
		assert.Same(t, ts2, head)

		err = c.Revert(ts2)
		require.NoError(t, err)

		head, err = c.Head()
		require.NoError(t, err)
		assert.Same(t, ts1, head)
	})

	t.Run("revert ring", func(t *testing.T) {
		c := NewTipSetCache(3)
		ts1 := mustMakeTs(nil, 1, dummyCid)
		_, err := c.Add(ts1)
		require.NoError(t, err)

		ts2 := mustMakeTs(nil, 2, dummyCid)
		_, err = c.Add(ts2)
		require.NoError(t, err)

		ts3 := mustMakeTs(nil, 3, dummyCid)
		_, err = c.Add(ts3)
		require.NoError(t, err)

		ts4 := mustMakeTs(nil, 4, dummyCid)
		_, err = c.Add(ts4)
		require.NoError(t, err)

		err = c.Revert(ts4)
		require.NoError(t, err)

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts3, head)
	})

	t.Run("zero sized cache", func(t *testing.T) {
		c := NewTipSetCache(0)

		ts1 := mustMakeTs(nil, 1, dummyCid)
		oldTail1, err := c.Add(ts1)
		assert.NoError(t, err)
		assert.Equal(t, 0, c.Len())
		assert.Same(t, ts1, oldTail1)

		ts, err := c.Head()
		assert.Error(t, err, ErrCacheEmpty)
		assert.Nil(t, ts)

		ts, err = c.Tail()
		assert.Error(t, err, ErrCacheEmpty)
		assert.Nil(t, ts)

		err = c.Revert(dummyTs)
		assert.Error(t, err, ErrEmptyRevert)
	})
}

func TestTipSetCacheAddOutOfOrder(t *testing.T) {
	c := NewTipSetCache(3)
	ts14 := mustMakeTs(nil, 14, dummyCid)
	_, err := c.Add(ts14)
	require.NoError(t, err)

	ts15 := mustMakeTs(nil, 15, dummyCid)
	_, err = c.Add(ts15)
	require.NoError(t, err)

	ts13 := mustMakeTs(nil, 13, dummyCid)
	_, err = c.Add(ts13)
	require.Error(t, err, ErrAddOutOfOrder)
}

func TestTipSetCacheRevertOutOfOrder(t *testing.T) {
	c := NewTipSetCache(3)
	ts14 := mustMakeTs(nil, 14, dummyCid)
	_, err := c.Add(ts14)
	require.NoError(t, err)

	ts15 := mustMakeTs(nil, 15, dummyCid)
	_, err = c.Add(ts15)
	require.NoError(t, err)

	err = c.Revert(ts14)
	require.Error(t, err, ErrRevertOutOfOrder)
}

func TestTipSetCacheSetCurrent(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewTipSetCache(3)

		ts1 := mustMakeTs(nil, 1, dummyCid)
		err := c.SetCurrent(ts1)
		assert.NoError(t, err)

		assert.Equal(t, 1, c.Len())
		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts1, head)
	})

	t.Run("same height", func(t *testing.T) {
		c := NewTipSetCache(3)

		ts14 := mustMakeTs(nil, 14, dummyCid)
		_, err := c.Add(ts14)
		require.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		ts14alt := mustMakeTs(nil, 14, dummyCid)
		err = c.SetCurrent(ts14alt)
		assert.NoError(t, err)

		assert.Equal(t, 1, c.Len())
		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts14alt, head)
	})

	t.Run("same tipset", func(t *testing.T) {
		c := NewTipSetCache(3)

		ts14 := mustMakeTs(nil, 14, dummyCid)
		_, err := c.Add(ts14)
		require.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		err = c.SetCurrent(ts14)
		assert.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts14, head)
	})

	t.Run("higher height", func(t *testing.T) {
		c := NewTipSetCache(3)

		ts14 := mustMakeTs(nil, 14, dummyCid)
		_, err := c.Add(ts14)
		require.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		ts16 := mustMakeTs(nil, 16, dummyCid)
		err = c.SetCurrent(ts16)
		assert.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts16, head)
	})

	t.Run("lower height reverts earlier", func(t *testing.T) {
		c := NewTipSetCache(3)

		ts14 := mustMakeTs(nil, 14, dummyCid)
		_, err := c.Add(ts14)
		require.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		ts15 := mustMakeTs(nil, 15, dummyCid)
		_, err = c.Add(ts15)
		require.NoError(t, err)
		assert.Equal(t, 2, c.Len())

		ts16 := mustMakeTs(nil, 16, dummyCid)
		_, err = c.Add(ts16)
		require.NoError(t, err)
		assert.Equal(t, 3, c.Len())

		ts15alt := mustMakeTs(nil, 15, dummyCid)
		err = c.SetCurrent(ts15alt)
		assert.NoError(t, err)
		assert.Equal(t, 2, c.Len()) // ts16 has been reverted, ts16 replaced by ts16alt

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts15alt, head)
	})

	t.Run("very low height reverts all", func(t *testing.T) {
		c := NewTipSetCache(3)

		ts14 := mustMakeTs(nil, 14, dummyCid)
		_, err := c.Add(ts14)
		require.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		ts15 := mustMakeTs(nil, 15, dummyCid)
		_, err = c.Add(ts15)
		require.NoError(t, err)
		assert.Equal(t, 2, c.Len())

		ts16 := mustMakeTs(nil, 16, dummyCid)
		_, err = c.Add(ts16)
		require.NoError(t, err)
		assert.Equal(t, 3, c.Len())

		ts12 := mustMakeTs(nil, 12, dummyCid)
		err = c.SetCurrent(ts12)
		assert.NoError(t, err)
		assert.Equal(t, 1, c.Len())

		head, err := c.Head()
		require.NoError(t, err)
		assert.Same(t, ts12, head)
	})
}

func TestTipSetCacheAddOnlyReturnsOldTailWhenFull(t *testing.T) {
	c := NewTipSetCache(3)
	ts14 := mustMakeTs(nil, 14, dummyCid)
	oldTail, err := c.Add(ts14)
	require.NoError(t, err)
	assert.Nil(t, oldTail)

	ts15 := mustMakeTs(nil, 15, dummyCid)
	oldTail, err = c.Add(ts15)
	require.NoError(t, err)
	assert.Nil(t, oldTail)

	ts15Revert := mustMakeTs(nil, 15, dummyCid)
	err = c.Revert(ts15Revert)
	require.NoError(t, err)

	ts15New := mustMakeTs(nil, 15, dummyCid)
	oldTail, err = c.Add(ts15New)
	require.NoError(t, err)
	assert.Nil(t, oldTail) // must be nil since cache is not full

	ts16 := mustMakeTs(nil, 16, dummyCid)
	oldTail, err = c.Add(ts16)
	require.NoError(t, err)
	assert.Nil(t, oldTail)

	ts17 := mustMakeTs(nil, 17, dummyCid)
	oldTail, err = c.Add(ts17)
	require.NoError(t, err)
	assert.Same(t, oldTail, ts14) // cache is now full so oldest is evicted
}

func TestTipSetCacheWarming(t *testing.T) {
	t.Run("happy path, warm with confidence", func(t *testing.T) {
		t4 := mustMakeTs(nil, 10, dummyCid)
		t3 := mustMakeTs(t4.Cids(), 11, dummyCid)
		t2 := mustMakeTs(t3.Cids(), 12, dummyCid)
		t1 := mustMakeTs(t2.Cids(), 13, dummyCid)
		head := mustMakeTs(t1.Cids(), 14, dummyCid)
		tw := &TestWarmer{
			tss: []*types.TipSet{t1, t2, t3, t4},
		}

		c := NewTipSetCache(4)

		err := c.Warm(context.Background(), head, tw.GetTipSet)
		assert.NoError(t, err)

		newHead := mustMakeTs(head.Cids(), 15, dummyCid)
		expectC4, err := c.Add(newHead)
		assert.NoError(t, err)
		assert.Equal(t, expectC4, t4)
	})

	t.Run("warming with zero confidence", func(t *testing.T) {
		head := mustMakeTs(nil, 14, dummyCid)
		tw := &TestWarmer{}
		c := NewTipSetCache(0)

		err := c.Warm(context.Background(), head, tw.GetTipSet)
		assert.NoError(t, err)

		newHead := mustMakeTs(head.Cids(), 15, dummyCid)
		expectHead, err := c.Add(newHead)
		assert.NoError(t, err)
		assert.Equal(t, expectHead, newHead)
	})

	t.Run("incomplete warm", func(t *testing.T) {
		t2 := mustMakeTs(nil, 0, dummyCid)
		t1 := mustMakeTs(t2.Cids(), 13, dummyCid)
		head := mustMakeTs(t1.Cids(), 14, dummyCid)
		tw := &TestWarmer{
			tss: []*types.TipSet{t1, t2},
		}

		c := NewTipSetCache(4)

		err := c.Warm(context.Background(), head, tw.GetTipSet)
		assert.NoError(t, err)

		expectNil, err := c.Add(mustMakeTs(head.Cids(), 15, dummyCid))
		assert.NoError(t, err)
		assert.Nil(t, expectNil)
		expectNil, err = c.Add(mustMakeTs(nil, 16, dummyCid))
		assert.NoError(t, err)
		assert.Nil(t, expectNil)
		expectT2, err := c.Add(mustMakeTs(nil, 17, dummyCid))
		assert.NoError(t, err)
		assert.Equal(t, t2, expectT2)
	})
}

type TestWarmer struct {
	idx int
	tss []*types.TipSet
}

func (tw *TestWarmer) GetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	defer func() {
		tw.idx += 1
	}()
	return tw.tss[tw.idx], nil
}

func mustMakeTs(parents []cid.Cid, h abi.ChainEpoch, msgcid cid.Cid) *types.TipSet {
	a, _ := address.NewFromString("t00")
	b, _ := address.NewFromString("t02")
	ts, err := types.NewTipSet([]*types.BlockHeader{
		{
			Height: h,
			Miner:  a,

			Parents: parents,

			Ticket: &types.Ticket{VRFProof: []byte{byte(h % 2)}},

			ParentStateRoot:       dummyCid,
			Messages:              msgcid,
			ParentMessageReceipts: dummyCid,

			BlockSig:     &crypto.Signature{Type: crypto.SigTypeBLS},
			BLSAggregate: &crypto.Signature{Type: crypto.SigTypeBLS},
		},
		{
			Height: h,
			Miner:  b,

			Parents: parents,

			Ticket: &types.Ticket{VRFProof: []byte{byte((h + 1) % 2)}},

			ParentStateRoot:       dummyCid,
			Messages:              msgcid,
			ParentMessageReceipts: dummyCid,

			BlockSig:     &crypto.Signature{Type: crypto.SigTypeBLS},
			BLSAggregate: &crypto.Signature{Type: crypto.SigTypeBLS},
		},
	})
	if err != nil {
		panic("mustMakeTs: " + err.Error())
	}

	return ts
}

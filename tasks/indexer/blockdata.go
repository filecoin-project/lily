package indexer

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	"github.com/filecoin-project/sentinel-visor/model/visor"
)

func NewUnindexedBlockData() *UnindexedBlockData {
	return &UnindexedBlockData{
		has: make(map[types.TipSetKey]struct{}),
	}
}

type UnindexedBlockData struct {
	has               map[types.TipSetKey]struct{}
	height            abi.ChainEpoch
	blks              blocks.BlockHeaders
	parents           blocks.BlockParents
	drandEntries      blocks.DrandEntries
	drandBlockEntries blocks.DrandBlockEntries
	tipsets           visor.ProcessingTipSetList
}

func (u *UnindexedBlockData) AddTipSet(ts *types.TipSet) {
	u.MarkSeen(ts.Key())
	if ts.Height() > u.height {
		u.height = ts.Height()
	}
	u.tipsets = append(u.tipsets, visor.NewProcessingTipSet(ts))
	for _, header := range ts.Blocks() {
		u.AddBlock(header)
	}
}

func (u *UnindexedBlockData) AddBlock(bh *types.BlockHeader) {
	u.blks = append(u.blks, blocks.NewBlockHeader(bh))
	u.parents = append(u.parents, blocks.NewBlockParents(bh)...)
	u.drandEntries = append(u.drandEntries, blocks.NewDrandEnties(bh)...)
	u.drandBlockEntries = append(u.drandBlockEntries, blocks.NewDrandBlockEntries(bh)...)
}

func (u *UnindexedBlockData) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "Indexer.PersistBlockData")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := u.blks.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persist block headers: %w", err)
		}

		if err := u.parents.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persist block parents: %w", err)
		}

		if err := u.drandEntries.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persist drand entries: %w", err)
		}

		if err := u.drandBlockEntries.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persist drand block entries: %w", err)
		}

		if err := u.tipsets.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persist processing tipsets: %w", err)
		}
		return nil
	})
}

func (u *UnindexedBlockData) Size() int {
	return len(u.tipsets)
}

func (u *UnindexedBlockData) Height() abi.ChainEpoch {
	return u.height
}

func (u *UnindexedBlockData) Seen(tsk types.TipSetKey) bool {
	_, has := u.has[tsk]
	return has
}

func (u *UnindexedBlockData) MarkSeen(tsk types.TipSetKey) {
	u.has[tsk] = struct{}{}
}

// Reset clears the unindexed data but keeps the history of which cids have been seen.
func (u *UnindexedBlockData) Reset() {
	u.blks = u.blks[:0]
	u.parents = u.parents[:0]
	u.drandEntries = u.drandEntries[:0]
	u.drandBlockEntries = u.drandBlockEntries[:0]
	u.tipsets = u.tipsets[:0]
}

package indexer

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"

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
	pstateChanges     visor.ProcessingStateChangeList
	pmessages         visor.ProcessingMessageList
}

func (u *UnindexedBlockData) AddTipSet(ts *types.TipSet) {
	u.MarkSeen(ts.Key())
	if ts.Height() > u.height {
		u.height = ts.Height()
	}
	u.pstateChanges = append(u.pstateChanges, visor.NewProcessingStateChange(ts))
	u.pmessages = append(u.pmessages, visor.NewProcessingMessage(ts))
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

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		grp, ctx := errgroup.WithContext(ctx)

		grp.Go(func() error {
			if err := u.blks.PersistWithTx(ctx, tx); err != nil {
				return xerrors.Errorf("persist block headers: %w", err)
			}
			return nil
		})

		grp.Go(func() error {
			if err := u.parents.PersistWithTx(ctx, tx); err != nil {
				return xerrors.Errorf("persist block parents: %w", err)
			}
			return nil
		})

		grp.Go(func() error {
			if err := u.drandEntries.PersistWithTx(ctx, tx); err != nil {
				return xerrors.Errorf("persist drand entries: %w", err)
			}
			return nil
		})

		grp.Go(func() error {
			if err := u.drandBlockEntries.PersistWithTx(ctx, tx); err != nil {
				return xerrors.Errorf("persist drand block entries: %w", err)
			}
			return nil
		})

		grp.Go(func() error {
			if err := u.pstateChanges.PersistWithTx(ctx, tx); err != nil {
				return xerrors.Errorf("persist processing state changes: %w", err)
			}
			return nil
		})

		grp.Go(func() error {
			if err := u.pmessages.PersistWithTx(ctx, tx); err != nil {
				return xerrors.Errorf("persist processing messages: %w", err)
			}
			return nil
		})

		if err := grp.Wait(); err != nil {
			log.Warnf("rolling back unindexed block data", "error", err)
			return err
		}

		return nil
	})
}

func (u *UnindexedBlockData) Size() int {
	return len(u.pstateChanges)
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
	u.pstateChanges = u.pstateChanges[:0]
	u.pmessages = u.pmessages[:0]
}

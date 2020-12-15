package indexer

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
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
	u.drandBlockEntries = append(u.drandBlockEntries, blocks.NewDrandBlockEntries(bh)...)
}

func (u *UnindexedBlockData) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "UnindexedBlockData.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if err := u.blks.Persist(ctx, s); err != nil {
		return xerrors.Errorf("persist block headers: %w", err)
	}

	if err := u.parents.Persist(ctx, s); err != nil {
		return xerrors.Errorf("persist block parents: %w", err)
	}

	if err := u.drandBlockEntries.Persist(ctx, s); err != nil {
		return xerrors.Errorf("persist drand block entries: %w", err)
	}

	if err := u.tipsets.Persist(ctx, s); err != nil {
		return xerrors.Errorf("persist processing tipsets: %w", err)
	}
	return nil
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
	u.drandBlockEntries = u.drandBlockEntries[:0]
	u.tipsets = u.tipsets[:0]
}

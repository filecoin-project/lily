package blocks

import (
	"context"
	"fmt"
	"golang.org/x/xerrors"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
)

func NewBlockSynced(header *types.BlockHeader) *BlockSynced {
	return &BlockSynced{
		Cid:      header.Cid().String(),
		Height:   int64(header.Height),
		SyncedAt: time.Now(),
	}
}

type BlockSynced struct {
	tableName struct{} `pg:"blocks_synced"`

	Cid         string    `pg:",pk,notnull"`
	Height      int64     `pg:",use_zero"`
	SyncedAt    time.Time `pg:",notnull"`
	ProcessedAt time.Time
	CompletedAt time.Time
}

func (bs *BlockSynced) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bs).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting block synced: %w", err)
	}
	return nil
}

type BlocksSynced []*BlockSynced

func (bss BlocksSynced) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		for _, bs := range bss {
			if err := bs.PersistWithTx(ctx, tx); err != nil {
				return fmt.Errorf("persist blocks synced: %v", err)
			}
		}
		return nil
	})
}

func (bss BlocksSynced) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, bs := range bss {
		if err := bs.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persist blocks synced: %v", err)
		}
	}
	return nil
}

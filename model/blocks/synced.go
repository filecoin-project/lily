package blocks

import (
	"context"
	"fmt"
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
	Height      int64     `pg:",notnull"`
	SyncedAt    time.Time `pg:",notnull"`
	ProcessedAt time.Time
}

func (bs *BlockSynced) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

type BlocksSynced []*BlockSynced

func (bss BlocksSynced) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}
	for _, bs := range bss {
		if err := bs.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persist blocks synced: %v", err)
		}
	}
	return tx.CommitContext(ctx)
}

func (bss BlocksSynced) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, bs := range bss {
		if err := bs.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persist blocks synced: %v", err)
		}
	}
	return nil
}

package blocks

import (
	"context"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
)

type BlockParent struct {
	Block  string `pg:",pk,notnull"`
	Parent string `pg:",notnull"`
}

func (bp *BlockParent) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

type BlockParents []*BlockParent

func NewBlockParents(header *types.BlockHeader) BlockParents {
	var out BlockParents
	for _, p := range header.Parents {
		out = append(out, &BlockParent{
			Block:  header.Cid().String(),
			Parent: p.String(),
		})
	}
	return out
}

func (bps BlockParents) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}
	for _, p := range bps {
		if err := p.PersistWithTx(ctx, tx); err != nil {
			return nil
		}
	}
	return tx.CommitContext(ctx)
}

func (bps BlockParents) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, p := range bps {
		if err := p.PersistWithTx(ctx, tx); err != nil {
			return nil
		}
	}
	return nil
}

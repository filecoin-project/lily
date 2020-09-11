package blocks

import (
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

type DrandEntrie struct {
	Round uint64 `pg:",pk,use_zero"`
	Data  []byte `pg:",notnull"`
}

func (de *DrandEntrie) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, de).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting drand entries: %w", err)
	}
	return nil
}

func NewDrandEnties(header *types.BlockHeader) DrandEntries {
	var out DrandEntries
	for _, ent := range header.BeaconEntries {
		out = append(out, &DrandEntrie{
			Round: ent.Round,
			Data:  ent.Data,
		})
	}
	return out
}

type DrandEntries []*DrandEntrie

func (des DrandEntries) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}

	for _, ent := range des {
		if err := ent.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return tx.CommitContext(ctx)
}

func (des DrandEntries) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, ent := range des {
		if err := ent.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persist drand entries: %v", err)
		}
	}
	return nil
}

func NewDrandBlockEntries(header *types.BlockHeader) DrandBlockEntries {
	var out DrandBlockEntries
	for _, ent := range header.BeaconEntries {
		out = append(out, &DrandBlockEntrie{
			Round: ent.Round,
			Block: header.Cid().String(),
		})
	}
	return out
}

type DrandBlockEntrie struct {
	Round uint64 `pg:",pk,use_zero"`
	Block string `pg:",notnull"`
}

func (dbe *DrandBlockEntrie) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, dbe).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting drand block entries: %w", err)
	}
	return nil
}

type DrandBlockEntries []*DrandBlockEntrie

func (dbes DrandBlockEntries) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}

	for _, ent := range dbes {
		if err := ent.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persist drand block entries: %w", err)
		}
	}
	return tx.CommitContext(ctx)
}

func (dbes DrandBlockEntries) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, ent := range dbes {
		if err := ent.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persist drand block entries: %w", err)
		}
	}
	return nil
}

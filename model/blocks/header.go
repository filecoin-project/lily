package blocks

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/types"
)

type BlockHeader struct {
	Cid             string `pg:",pk,notnull"`
	Miner           string `pg:",notnull"`
	ParentWeight    string `pg:",notnull"`
	ParentBaseFee   string `pg:",notnull"`
	ParentStateRoot string `pg:",notnull"`

	Height        int64  `pg:",use_zero"`
	WinCount      int64  `pg:",use_zero"`
	Timestamp     uint64 `pg:",use_zero"`
	ForkSignaling uint64 `pg:",use_zero"`

	Ticket        []byte
	ElectionProof []byte
}

func NewBlockHeader(bh *types.BlockHeader) *BlockHeader {
	return &BlockHeader{
		Cid:             bh.Cid().String(),
		Miner:           bh.Miner.String(),
		ParentWeight:    bh.ParentWeight.String(),
		ParentBaseFee:   bh.ParentBaseFee.String(),
		ParentStateRoot: bh.ParentStateRoot.String(),
		Height:          int64(bh.Height),
		WinCount:        bh.ElectionProof.WinCount,
		Timestamp:       bh.Timestamp,
		ForkSignaling:   bh.ForkSignaling,
		Ticket:          bh.Ticket.VRFProof,
		ElectionProof:   bh.ElectionProof.VRFProof,
	}
}

func (bh *BlockHeader) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bh).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting block header: %w", err)
	}
	return nil
}

type BlockHeaders []*BlockHeader

func (bh BlockHeaders) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}

	for _, h := range bh {
		if err := h.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persist headers: %v", err)
		}
	}
	return tx.CommitContext(ctx)
}

func (bh BlockHeaders) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, h := range bh {
		if err := h.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persist headers: %v", err)
		}
	}
	return nil
}

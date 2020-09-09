package blocks

import (
	"context"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
)

type BlockHeaders map[cid.Cid]*BlockHeader

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
	//for _, header := range headers {
	//	if _, err := tx.ModelContext(ctx, NewBlockHeader(header)).
	//		OnConflict("do nothing").
	//		Insert(); err != nil {
	//		return err
	//	}
	//}
	return tx.CommitContext(ctx)
}

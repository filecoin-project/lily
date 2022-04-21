package indexer

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
)

type Indexer interface {
	TipSet(ctx context.Context, ts *types.TipSet, priority string, tasks ...string) (bool, error)
}

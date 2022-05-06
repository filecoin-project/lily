package views

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lily/wait"
)

var chainVisViews = []string{
	"chain_visualizer_blocks_view",
	"chain_visualizer_blocks_with_parents_view",
	"chain_visualizer_chain_data_view",
	"chain_visualizer_orphans_view",
	"derived_consensus_chain_view",
}

func NewChainVisRefresher(d *storage.Database, refreshRate time.Duration) *ChainVisRefresher {
	return &ChainVisRefresher{
		db:          d,
		refreshRate: refreshRate,
	}
}

// ChainVisRefresher is a task which refreshes a set of views that support
// chain visualization queries at a specific refreshRate
type ChainVisRefresher struct {
	db          *storage.Database
	refreshRate time.Duration
}

// Run starts regularly refreshing until context is done or an error occurs
func (r *ChainVisRefresher) Run(ctx context.Context) error {
	if r.refreshRate == 0 {
		return nil
	}
	return wait.RepeatUntil(ctx, r.refreshRate, r.refreshView)
}

func (r *ChainVisRefresher) refreshView(ctx context.Context) (bool, error) {
	for _, v := range chainVisViews {
		_, err := r.db.ExecContext(ctx, fmt.Sprintf("REFRESH MATERIALIZED VIEW %s;", v))
		if err != nil {
			return true, fmt.Errorf("refresh %s: %w", v, err)
		}
	}
	return false, nil
}

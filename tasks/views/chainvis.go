package views

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
)

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
	_, err := r.db.DB.ExecContext(ctx, "REFRESH MATERIALIZED VIEW chain_visualizer_blocks_view;")
	if err != nil {
		return true, xerrors.Errorf("refresh chain_visualizer_blocks_view: %w", err)
	}
	_, err = r.db.DB.ExecContext(ctx, "REFRESH MATERIALIZED VIEW chain_visualizer_blocks_with_parents_view;")
	if err != nil {
		return true, xerrors.Errorf("refresh chain_visualizer_blocks_with_parents_view: %w", err)
	}
	_, err = r.db.DB.ExecContext(ctx, "REFRESH MATERIALIZED VIEW chain_visualizer_orphans_view;")
	if err != nil {
		return true, xerrors.Errorf("refresh chain_visualizer_orphans_view: %w", err)
	}
	_, err = r.db.DB.ExecContext(ctx, "REFRESH MATERIALIZED VIEW chain_visualizer_chain_data_view;")
	if err != nil {
		return true, xerrors.Errorf("refresh chain_visualizer_chain_data_view: %w", err)
	}
	return false, nil
}

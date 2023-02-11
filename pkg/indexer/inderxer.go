package indexer

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/pkg/extract"
	"github.com/filecoin-project/lily/tasks"
)

func NewStateIndexer(api tasks.DataSource, handler StateHandler) *StateIndexer {
	return &StateIndexer{
		api:     api,
		handler: handler,
	}
}

type StateIndexer struct {
	api     tasks.DataSource
	handler StateHandler
}

type StateHandler interface {
	Persist(ctx context.Context, chainState *extract.ChainState) error
}

func (s *StateIndexer) TipSet(ctx context.Context, ts *types.TipSet, opts ...indexer.Option) (bool, error) {
	parent, err := s.api.TipSet(ctx, ts.Parents())
	if err != nil {
		return false, err
	}

	chainState, err := extract.State(ctx, s.api, ts, parent)
	if err != nil {
		return false, err
	}

	if err := s.handler.Persist(ctx, chainState); err != nil {
		return false, err
	}
	return true, nil
}

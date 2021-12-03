package chain

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens"
)

func NewWalker(obs TipSetObserver, node lens.API, minHeight, maxHeight int64) *Walker {
	return &Walker{
		node:      node,
		obs:       obs,
		minHeight: minHeight,
		maxHeight: maxHeight,
	}
}

// Walker is a job that indexes blocks by walking the chain history.
type Walker struct {
	node      lens.API
	obs       TipSetObserver
	minHeight int64 // limit persisting to tipsets equal to or above this height
	maxHeight int64 // limit persisting to tipsets equal to or below this height}
	done      chan struct{}
}

// Run starts walking the chain history and continues until the context is done or
// the start of the chain is reached.
func (c *Walker) Run(ctx context.Context) error {
	c.done = make(chan struct{})
	defer func() {
		close(c.done)
		if err := c.obs.Close(); err != nil {
			log.Errorw("walker failed to close TipSetObserver", "error", err)
		}
	}()

	ts, err := c.node.ChainHead(ctx)
	if err != nil {
		return xerrors.Errorf("get chain head: %w", err)
	}

	if int64(ts.Height()) < c.minHeight {
		return xerrors.Errorf("cannot walk history, chain head (%d) is earlier than minimum height (%d)", int64(ts.Height()), c.minHeight)
	}

	// Start at maxHeight+1 so that the tipset at maxHeight becomes the parent for any tasks that need to make a diff between two tipsets.
	// A walk where min==max must still process two tipsets to be sure of extracting data.
	if int64(ts.Height()) > c.maxHeight+1 {
		ts, err = c.node.ChainGetTipSetAfterHeight(ctx, abi.ChainEpoch(c.maxHeight+1), types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("get tipset by height: %w", err)
		}
	}

	if err := c.WalkChain(ctx, c.node, ts); err != nil {
		return xerrors.Errorf("walk chain: %w", err)
	}

	return nil
}

func (c *Walker) Done() <-chan struct{} {
	return c.done
}

func (c *Walker) WalkChain(ctx context.Context, node lens.API, ts *types.TipSet) error {
	ctx, span := otel.Tracer("").Start(ctx, "Walker.WalkChain", trace.WithAttributes(
		attribute.Int64("tipset_height", int64(ts.Height())),
		attribute.String("tipset", ts.String()),
		attribute.Int64("min_height", c.minHeight),
		attribute.Int64("max_height", c.maxHeight),
	))
	defer span.End()

	log.Debugw("found tipset", "height", ts.Height())
	if err := c.obs.TipSet(ctx, ts); err != nil {
		span.RecordError(err)
		return xerrors.Errorf("notify tipset: %w", err)
	}

	var err error
	for ts.Height() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ts, err = node.ChainGetTipSet(ctx, ts.Parents())
		if err != nil {
			span.RecordError(err)
			return xerrors.Errorf("get tipset: %w", err)
		}

		if int64(ts.Height()) < c.minHeight {
			break
		}

		log.Debugw("found tipset", "height", ts.Height())
		if err := c.obs.TipSet(ctx, ts); err != nil {
			span.RecordError(err)
			return xerrors.Errorf("notify tipset: %w", err)
		}

	}

	return nil
}

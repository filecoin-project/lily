package walk

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/lens"
)

var log = logging.Logger("lily/chain/walk")

func NewWalker(obs indexer.Indexer, node lens.API, name string, tasks []string, minHeight, maxHeight int64) *Walker {
	return &Walker{
		node:      node,
		obs:       obs,
		name:      name,
		tasks:     tasks,
		minHeight: minHeight,
		maxHeight: maxHeight,
	}
}

// Walker is a job that indexes blocks by walking the chain history.
type Walker struct {
	node      lens.API
	obs       indexer.Indexer
	name      string
	tasks     []string
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
	}()

	head, err := c.node.ChainHead(ctx)
	if err != nil {
		return xerrors.Errorf("get chain head: %w", err)
	}

	if int64(head.Height()) < c.minHeight {
		return xerrors.Errorf("cannot walk history, chain head (%d) is earlier than minimum height (%d)", int64(head.Height()), c.minHeight)
	}

	start := head
	// Start at maxHeight+1 so that the tipset at maxHeight becomes the parent for any tasks that need to make a diff between two tipsets.
	// A walk where min==max must still process two tipsets to be sure of extracting data.
	if int64(head.Height()) > c.maxHeight+1 {
		start, err = c.node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(c.maxHeight), head.Key())
		if err != nil {
			return xerrors.Errorf("get tipset by height: %w", err)
		}
	}

	if err := c.WalkChain(ctx, c.node, start); err != nil {
		return xerrors.Errorf("walk chain: %w", err)
	}

	return nil
}

func (c *Walker) Done() <-chan struct{} {
	return c.done
}

func (c *Walker) WalkChain(ctx context.Context, node lens.API, ts *types.TipSet) error {
	ctx, span := otel.Tracer("").Start(ctx, "Walker.WalkChain")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("tipset", ts.String()),
			attribute.Int64("min_height", c.minHeight),
			attribute.Int64("max_height", c.maxHeight),
		)
	}
	defer span.End()

	var err error
	for int64(ts.Height()) >= c.minHeight && ts.Height() != 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		log.Infow("walk tipset", "height", ts.Height(), "reporter", c.name)
		if success, err := c.obs.TipSet(ctx, ts, indexer.WithIndexerType(indexer.Walk), indexer.WithTasks(c.tasks)); err != nil {
			span.RecordError(err)
			return xerrors.Errorf("notify tipset: %w", err)
		} else if !success {
			log.Errorw("walk incomplete", "height", ts.Height(), "tipset", ts.Key().String(), "reporter", c.name)
		}
		log.Infow("walk tipset success", "height", ts.Height(), "reporter", c.name)

		ts, err = node.ChainGetTipSet(ctx, ts.Parents())
		if err != nil {
			span.RecordError(err)
			return xerrors.Errorf("get tipset: %w", err)
		}
	}

	return nil
}

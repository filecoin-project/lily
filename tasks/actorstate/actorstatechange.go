package actorstate

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"github.com/raulk/clock"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
)

const idleSleepInterval = 60 * time.Second // time to wait if the processor runs out of blocks to process

func NewActorStateChangeProcessor(d *storage.Database, node lens.API, leaseLength time.Duration, batchSize int, minHeight, maxHeight int64) *ActorStateChangeProcessor {
	return &ActorStateChangeProcessor{
		node:        node,
		storage:     d,
		leaseLength: leaseLength,
		batchSize:   batchSize,
		minHeight:   minHeight,
		maxHeight:   maxHeight,
		clock:       clock.New(),
	}
}

// ActorStateChangeProcessor is a task that processes blocks to detect actors whose states have changed and persists
// their details to the database.
type ActorStateChangeProcessor struct {
	node        lens.API
	storage     *storage.Database
	leaseLength time.Duration // length of time to lease work for
	batchSize   int           // number of blocks to lease in a batch
	minHeight   int64         // limit processing to tipsets equal to or above this height
	maxHeight   int64         // limit processing to tipsets equal to or below this height
	clock       clock.Clock
}

// Run starts processing batches of blocks and blocks until the context is done or
// an error occurs.
func (p *ActorStateChangeProcessor) Run(ctx context.Context) error {
	// Loop until context is done or processing encounters a fatal error
	return wait.RepeatUntil(ctx, batchInterval, p.processBatch)
}

func (p *ActorStateChangeProcessor) processBatch(ctx context.Context) (bool, error) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "statechange"))
	ctx, span := global.Tracer("").Start(ctx, "ActorStateChangeProcessor.processBatch")
	defer span.End()

	claimUntil := p.clock.Now().Add(p.leaseLength)

	// Lease some tipsets to work on
	batch, err := p.storage.LeaseStateChanges(ctx, claimUntil, p.batchSize, p.minHeight, p.maxHeight)
	if err != nil {
		return true, err
	}

	// If we have no tipsets to work on then wait before trying again
	if len(batch) == 0 {
		sleepInterval := wait.Jitter(idleSleepInterval, 2)
		log.Debugf("no tipsets to process, waiting for %s", sleepInterval)
		time.Sleep(sleepInterval)
		return false, nil
	}

	log.Debugw("leased batch of tipsets", "count", len(batch))
	ctx, cancel := context.WithDeadline(ctx, claimUntil)
	defer cancel()

	for _, item := range batch {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return false, nil // Don't propagate cancelation error so we can resume processing cleanly
		default:
		}

		errorLog := log.With("height", item.Height, "tipset", item.TipSet)

		if err := p.processItem(ctx, item); err != nil {
			errorLog.Errorw("failed to process tipset", "error", err.Error())
			if err := p.storage.MarkStateChangeComplete(ctx, item.TipSet, item.Height, p.clock.Now(), err.Error()); err != nil {
				errorLog.Errorw("failed to mark tipset complete", "error", err.Error())
			}
			continue
		}

		if err := p.storage.MarkStateChangeComplete(ctx, item.TipSet, item.Height, p.clock.Now(), ""); err != nil {
			errorLog.Errorw("failed to mark tipset complete", "error", err.Error())
		}
	}

	return false, nil
}

func (p *ActorStateChangeProcessor) processItem(ctx context.Context, item *visor.ProcessingTipSet) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorStateChangeProcessor.processItem")
	defer span.End()
	span.SetAttributes(label.Any("height", item.Height), label.Any("tipset", item.TipSet))

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	tsk, err := item.TipSetKey()
	if err != nil {
		return xerrors.Errorf("get tipsetkey: %w", err)
	}

	ts, err := p.node.ChainGetTipSet(ctx, tsk)
	if err != nil {
		return xerrors.Errorf("get tipset: %w", err)
	}

	if item.Height > 0 {
		if err := p.processTipSet(ctx, ts); err != nil {
			return xerrors.Errorf("process tipset: %w", err)
		}
	} else {
		gp := NewGenesisProcessor(p.storage, p.node)
		if err := gp.ProcessGenesis(ctx, ts); err != nil {
			return xerrors.Errorf("process genesis: %w", err)
		}
	}

	return nil
}

func (p *ActorStateChangeProcessor) processTipSet(ctx context.Context, ts *types.TipSet) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorStateChangeProcessor.processTipSet")
	defer span.End()
	ll := log.With("height", int64(ts.Height()))

	ll.Debugw("processing tipset")

	pts, err := p.node.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return xerrors.Errorf("get parent tipset: %w", err)
	}

	changes, err := p.node.StateChangedActors(ctx, pts.ParentState(), ts.ParentState())
	if err != nil {
		return xerrors.Errorf("get actor changes: %w", err)
	}

	ll.Debugw("found actor state changes", "count", len(changes))

	var palist visor.ProcessingActorList

	for str, act := range changes {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		addr, err := address.NewFromString(str)
		if err != nil {
			return xerrors.Errorf("parse address: %w", err)
		}

		_, err = p.node.StateGetActor(ctx, addr, pts.Key())
		if err != nil {
			if strings.Contains(err.Error(), "actor not found") {
				ll.Debugw("actor not found", "addr", str)
				// TODO consider tracking deleted actors
				continue
			}
			return xerrors.Errorf("get actor: %w", err)
		}

		_, err = p.node.StateGetActor(ctx, addr, pts.Parents())
		if err != nil {
			if strings.Contains(err.Error(), "actor not found") {
				ll.Debugw("parent actor not found", "addr", str)
				// TODO consider tracking deleted actors
				continue
			}
			return xerrors.Errorf("get actor parent: %w", err)
		}

		palist = append(palist, &visor.ProcessingActor{
			Head:            act.Head.String(),
			Code:            act.Code.String(),
			Nonce:           strconv.FormatUint(act.Nonce, 10),
			Balance:         act.Balance.String(),
			Address:         addr.String(),
			TipSet:          pts.Key().String(),
			ParentTipSet:    pts.Parents().String(),
			ParentStateRoot: pts.ParentState().String(),
			Height:          int64(ts.Height()),
			AddedAt:         p.clock.Now(),
		})
	}

	ll.Debugw("persisting tipset", "state_changes", len(palist))
	if err := p.storage.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return palist.PersistWithTx(ctx, tx)
	}); err != nil {
		return xerrors.Errorf("persist: %w", err)
	}

	return nil

}

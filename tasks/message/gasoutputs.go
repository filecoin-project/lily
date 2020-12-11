package message

import (
	"context"
	"errors"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/raulk/clock"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model/derived"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/wait"
)

func NewGasOutputsProcessor(d *storage.Database, opener lens.APIOpener, leaseLength time.Duration, batchSize int, minHeight, maxHeight int64, useLeases bool) *GasOutputsProcessor {
	return &GasOutputsProcessor{
		opener:      opener,
		storage:     d,
		leaseLength: leaseLength,
		batchSize:   batchSize,
		minHeight:   minHeight,
		maxHeight:   maxHeight,
		clock:       clock.New(),
		useLeases:   useLeases,
	}
}

// GasOutputsProcessor is a task that processes messages with receipts to determine gas outputs.
type GasOutputsProcessor struct {
	opener      lens.APIOpener
	storage     *storage.Database
	leaseLength time.Duration // length of time to lease work for
	batchSize   int           // number of messages to lease in a batch
	minHeight   int64         // limit processing to messages from tipsets equal to or above this height
	maxHeight   int64         // limit processing to messages from tipsets equal to or below this height
	clock       clock.Clock
	useLeases   bool // when true this task will update the claimed_until column in the processing table (which can cause contention)
}

// Run starts processing batches of messages until the context is done or
// an error occurs.
func (p *GasOutputsProcessor) Run(ctx context.Context) error {
	node, closer, err := p.opener.Open(ctx)
	if err != nil {
		return xerrors.Errorf("open lens: %w", err)
	}
	defer closer()

	// Loop until context is done or processing encounters a fatal error
	return wait.RepeatUntil(ctx, batchInterval, func(ctx context.Context) (bool, error) {
		return p.processBatch(ctx, node)
	})
}

func (p *GasOutputsProcessor) processBatch(ctx context.Context, node lens.API) (bool, error) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "gasoutputs"))
	ctx, span := global.Tracer("").Start(ctx, "GasOutputsProcessor.processBatch")
	defer span.End()

	claimUntil := p.clock.Now().Add(p.leaseLength)

	var batch []*derived.GasOutputs
	var err error

	if p.useLeases {
		// Lease some messages with receipts to work on
		batch, err = p.storage.LeaseGasOutputsMessages(ctx, claimUntil, p.batchSize, p.minHeight, p.maxHeight)
	} else {
		batch, err = p.storage.FindGasOutputsMessages(ctx, p.batchSize, p.minHeight, p.maxHeight)
	}

	if err != nil {
		return false, err
	}

	// If we have no messages to work on then wait before trying again
	if len(batch) == 0 {
		sleepInterval := wait.Jitter(idleSleepInterval, 2)
		log.Debugf("no messages to process, waiting for %s", sleepInterval)
		time.Sleep(sleepInterval)
		return false, nil
	}

	log.Debugw("processing batch of messages", "count", len(batch))
	if p.useLeases {
		var cancel func()
		ctx, cancel = context.WithDeadline(ctx, claimUntil)
		defer cancel()
	}

	for _, item := range batch {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return false, nil // Don't propagate cancelation error so we can resume processing cleanly
		default:
		}

		errorLog := log.With("cid", item.Cid)

		if err := p.processItem(ctx, node, item); err != nil {
			// Any errors are likely to be problems using the lens, mark this tipset as failed and exit this batch
			errorLog.Errorw("failed to process message", "error", err.Error())
			if err := p.storage.MarkGasOutputsMessagesComplete(ctx, item.Height, item.Cid, p.clock.Now(), err.Error()); err != nil {
				errorLog.Errorw("failed to mark message complete", "error", err.Error())
			}
			return false, xerrors.Errorf("process item: %w", err)
		}

		if err := p.storage.MarkGasOutputsMessagesComplete(ctx, item.Height, item.Cid, p.clock.Now(), ""); err != nil {
			errorLog.Errorw("failed to mark message complete", "error", err.Error())
		}
	}

	return false, nil
}

func (p *GasOutputsProcessor) processItem(ctx context.Context, node lens.API, item *derived.GasOutputs) error {
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	// Note: this item will only be processed if there are receipts for
	// it, which means there should be a tipset at height+1.  This is only
	// used to get the destination actor code, so we don't care about side
	// chains.
	child, err := node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(item.Height+1), types.NewTipSetKey())
	if err != nil {
		return xerrors.Errorf("Failed to load child tipset: %w", err)
	}

	st, err := state.LoadStateTree(node.Store(), child.ParentState())
	if err != nil {
		return xerrors.Errorf("load state tree when gas outputs for %s: %w", item.Cid, err)
	}

	dstAddr, err := address.NewFromString(item.To)
	if err != nil {
		return xerrors.Errorf("parse to address failed for gas outputs in %s: %w", item.Cid, err)
	}

	var dstActorCode cid.Cid
	dstActor, err := st.GetActor(dstAddr)
	if err != nil {
		if !errors.Is(err, types.ErrActorNotFound) {
			return xerrors.Errorf("get destination actor for gas outputs %s failed: %w", item.Cid, err)
		}
	} else {
		dstActorCode = dstActor.Code
	}

	baseFee, err := big.FromString(item.ParentBaseFee)
	if err != nil {
		return xerrors.Errorf("parse fee cap: %w", err)
	}
	feeCap, err := big.FromString(item.GasFeeCap)
	if err != nil {
		return xerrors.Errorf("parse fee cap: %w", err)
	}
	gasPremium, err := big.FromString(item.GasPremium)
	if err != nil {
		return xerrors.Errorf("parse gas premium: %w", err)
	}

	// this is here because the lotus.vm package doesn't take a context for this api call.
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ComputeGasOutputs"))
	cgoStop := metrics.Timer(ctx, metrics.LensRequestDuration)
	outputs := node.ComputeGasOutputs(item.GasUsed, item.GasLimit, baseFee, feeCap, gasPremium)
	cgoStop()

	item.ActorName = actorstate.ActorNameByCode(dstActorCode)
	item.BaseFeeBurn = outputs.BaseFeeBurn.String()
	item.OverEstimationBurn = outputs.OverEstimationBurn.String()
	item.MinerPenalty = outputs.MinerPenalty.String()
	item.MinerTip = outputs.MinerTip.String()
	item.Refund = outputs.Refund.String()
	item.GasRefund = outputs.GasRefund
	item.GasBurned = outputs.GasBurned

	if err := p.storage.PersistBatch(ctx, item); err != nil {
		return xerrors.Errorf("persisting gas outputs: %w", err)
	}

	return nil
}

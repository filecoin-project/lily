package message

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/go-pg/pg/v10"
	"github.com/raulk/clock"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model/derived"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
)

func NewGasOutputsProcessor(d *storage.Database, opener lens.APIOpener, leaseLength time.Duration, batchSize int, minHeight, maxHeight int64) *GasOutputsProcessor {
	return &GasOutputsProcessor{
		opener:      opener,
		storage:     d,
		leaseLength: leaseLength,
		batchSize:   batchSize,
		minHeight:   minHeight,
		maxHeight:   maxHeight,
		clock:       clock.New(),
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

	// Lease some messages with receipts to work on
	batch, err := p.storage.LeaseGasOutputsMessages(ctx, claimUntil, p.batchSize, p.minHeight, p.maxHeight)
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

	log.Debugw("leased batch of messages", "count", len(batch))
	ctx, cancel := context.WithDeadline(ctx, claimUntil)
	defer cancel()

	for _, item := range batch {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return false, nil // Don't propagate cancelation error so we can resume processing cleanly
		default:
		}

		errorLog := log.With("cid", item.Cid)

		if err := p.processItem(ctx, node, &item.GasOutputs); err != nil {
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

	item.BaseFeeBurn = outputs.BaseFeeBurn.String()
	item.OverEstimationBurn = outputs.OverEstimationBurn.String()
	item.MinerPenalty = outputs.MinerPenalty.String()
	item.MinerTip = outputs.MinerTip.String()
	item.Refund = outputs.Refund.String()
	item.GasRefund = outputs.GasRefund
	item.GasBurned = outputs.GasBurned

	if err := p.storage.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return item.PersistWithTx(ctx, tx)
	}); err != nil {
		return xerrors.Errorf("persisting gas outputs: %w", err)
	}

	return nil
}

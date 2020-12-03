package chain

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	logging "github.com/ipfs/go-log/v2"
	"github.com/raulk/clock"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	chainmodel "github.com/filecoin-project/sentinel-visor/model/chain"
	"github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/wait"
)

const (
	idleSleepInterval = 60 * time.Second       // time to wait if the processor runs out of blocks to process
	batchInterval     = 100 * time.Millisecond // time to wait between batches
)

var log = logging.Logger("chain")

type EconomicsStorage interface {
	RunInTransaction(ctx context.Context, fn func(tx *pg.Tx) error) error
	MarkTipSetEconomicsComplete(ctx context.Context, tipset string, height int64, completedAt time.Time, errorsDetected string) error
	LeaseTipSetEconomics(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingTipSetList, error)
}

func NewChainEconomicsProcessor(d EconomicsStorage, opener lens.APIOpener, leaseLength time.Duration, batchSize int, minHeight, maxHeight int64) *ChainEconomics {
	return &ChainEconomics{
		opener:      opener,
		storage:     d,
		leaseLength: leaseLength,
		batchSize:   batchSize,
		minHeight:   minHeight,
		maxHeight:   maxHeight,
		clock:       clock.New(),
	}
}

// ChainEconomics is a task that processes tipsets to calculate the circulating supply of Filecoin
// persists the results to the database.
type ChainEconomics struct {
	opener      lens.APIOpener
	storage     EconomicsStorage
	leaseLength time.Duration // length of time to lease work for
	batchSize   int           // number of tipsets to lease in a batch
	minHeight   int64         // limit processing to tipsets equal to or above this height
	maxHeight   int64         // limit processing to tipsets equal to or below this height
	clock       clock.Clock
}

// Run starts processing batches of tipsets until the context is done or
// an error occurs.
func (p *ChainEconomics) Run(ctx context.Context) error {
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

func (p *ChainEconomics) processBatch(ctx context.Context, node lens.API) (bool, error) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "chain/economics"))
	ctx, span := global.Tracer("").Start(ctx, "ChainEconomics.processBatch")
	defer span.End()

	claimUntil := p.clock.Now().Add(p.leaseLength)

	// Lease some blocks to work on
	batch, err := p.storage.LeaseTipSetEconomics(ctx, claimUntil, p.batchSize, p.minHeight, p.maxHeight)
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

		if err := p.processItem(ctx, node, item); err != nil {
			// Any errors are likely to be problems using the lens, mark this tipset as failed and exit this batch
			log.Errorw("failed to process tipset", "error", err.Error(), "height", item.Height)
			if err := p.storage.MarkTipSetEconomicsComplete(ctx, item.TipSet, item.Height, p.clock.Now(), err.Error()); err != nil {
				log.Errorw("failed to mark tipset economics complete", "error", err.Error(), "height", item.Height)
			}
			return false, xerrors.Errorf("process item: %w", err)
		}

		if err := p.storage.MarkTipSetEconomicsComplete(ctx, item.TipSet, item.Height, p.clock.Now(), ""); err != nil {
			log.Errorw("failed to mark tipset economics complete", "error", err.Error(), "height", item.Height)
		}
	}

	return false, nil
}

type EconomicsProcessItemsLens interface {
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)
	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (api.CirculatingSupply, error)
}

func (p *ChainEconomics) processItem(ctx context.Context, node EconomicsProcessItemsLens, item *visor.ProcessingTipSet) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainEconomics.processItem")
	defer span.End()
	span.SetAttributes(label.Any("height", item.Height), label.Any("tipset", item.TipSet))

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	tsk, err := item.TipSetKey()
	if err != nil {
		return xerrors.Errorf("get tipsetkey: %w", err)
	}

	ce, err := extractChainEconomicsModel(ctx, node, tsk)
	if err != nil {
		return xerrors.Errorf("extracting chain economics model: %w", err)
	}

	log.Debugw("persisting tipset", "height", item.Height)

	if err := p.storage.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ce.PersistWithTx(ctx, tx)
	}); err != nil {
		return xerrors.Errorf("persist: %w", err)
	}

	return nil
}

func extractChainEconomicsModel(ctx context.Context, node EconomicsProcessItemsLens, tsk types.TipSetKey) (*chainmodel.ChainEconomics, error) {
	ts, err := node.ChainGetTipSet(ctx, tsk)
	if err != nil {
		return nil, xerrors.Errorf("get tipset: %w", err)
	}

	supply, err := node.StateVMCirculatingSupplyInternal(ctx, tsk)
	if err != nil {
		return nil, xerrors.Errorf("get circulating supply: %w", err)
	}

	return &chainmodel.ChainEconomics{
		ParentStateRoot: ts.ParentState().String(),
		VestedFil:       supply.FilVested.String(),
		MinedFil:        supply.FilMined.String(),
		BurntFil:        supply.FilBurnt.String(),
		LockedFil:       supply.FilLocked.String(),
		CirculatingFil:  supply.FilCirculating.String(),
	}, nil
}

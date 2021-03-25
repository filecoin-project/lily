package commands

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	store "github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/sentinel-visor/schedule"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var Watch = &cli.Command{
	Name:  "watch",
	Usage: "Watch the head of the filecoin blockchain and process blocks as they arrive.",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:    "indexhead-confidence",
			Usage:   "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database",
			Value:   2,
			EnvVars: []string{"VISOR_INDEXHEAD_CONFIDENCE"},
		},
		&cli.StringFlag{
			Name:    "tasks",
			Usage:   "Comma separated list of tasks to run. Each task is reported separately in the database.",
			Value:   strings.Join([]string{chain.BlocksTask, chain.MessagesTask, chain.ChainEconomicsTask, chain.ActorStatesRawTask}, ","),
			EnvVars: []string{"VISOR_WATCH_TASKS"},
		},
	},
	Action: watch,
}

func watch(cctx *cli.Context) error {
	tasks := strings.Split(cctx.String("tasks"), ",")

	if err := setupLogging(cctx); err != nil {
		return xerrors.Errorf("setup logging: %w", err)
	}

	if err := setupMetrics(cctx); err != nil {
		return xerrors.Errorf("setup metrics: %w", err)
	}

	tcloser, err := setupTracing(cctx)
	if err != nil {
		return xerrors.Errorf("setup tracing: %w", err)
	}
	defer tcloser()

	lensOpener, lensCloser, err := setupLens(cctx)
	if err != nil {
		return xerrors.Errorf("setup lens: %w", err)
	}
	defer func() {
		lensCloser()
	}()

	var storage model.Storage = &storage.NullStorage{}
	if cctx.String("db") == "" {
		log.Warnw("database not specified, data will not be persisted")
	} else {
		db, err := setupDatabase(cctx)
		if err != nil {
			return xerrors.Errorf("setup database: %w", err)
		}
		storage = db
	}

	tsIndexer, err := chain.NewTipSetIndexer(lensOpener, storage, builtin.EpochDurationSeconds*time.Second, cctx.String("name"), tasks)
	if err != nil {
		return xerrors.Errorf("setup indexer: %w", err)
	}

	notifier := NewLotusChainNotifier(lensOpener)

	// TODO scheduler does not respect the ordering of these jobs, make it respect jobID when starting.
	// Subscribe to chain head events to be passed to the watcher
	scheduler := schedule.NewScheduler(cctx.Duration("task-delay"), &schedule.JobConfig{
		Name:                "ChainHeadNotifier",
		Job:                 notifier,
		RestartOnFailure:    true,
		RestartOnCompletion: true, // we always want the notifier to be running
		RestartDelay:        time.Minute,
	}, &schedule.JobConfig{
		Name: "Watcher",
		Job:  chain.NewWatcher(tsIndexer, notifier, cctx.Int("indexhead-confidence")),
		// TODO: add locker
		// Locker:              NewGlobalSingleton(ChainHeadIndexerLockID, rctx.db), // only want one forward indexer anywhere to be running
		RestartOnFailure:    true,
		RestartOnCompletion: true, // we always want the indexer to be running
		RestartDelay:        time.Minute,
	})

	// Start the scheduler and wait for it to complete or to be cancelled.
	err = scheduler.Run(cctx.Context)
	if !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// LotusChainNotifier is a head event notifier that subscribes to a lens's ChainNotify method and adapts the
// events received for use by a chain.Watcher
// NOTE: this functionality will be probably folded into the Lotus API lens since other lenses will support more
// direct methods of accessing new tipsets
type LotusChainNotifier struct {
	opener lens.APIOpener

	mu     sync.Mutex            // protects following fields
	events chan *chain.HeadEvent // created lazily, closed by first cancel call
	err    error                 // set to non-nil by the first cancel call
}

func NewLotusChainNotifier(opener lens.APIOpener) *LotusChainNotifier {
	return &LotusChainNotifier{
		opener: opener,
	}
}

func (c *LotusChainNotifier) eventsCh() chan *chain.HeadEvent {
	// caller must hold mu
	if c.events == nil {
		c.events = make(chan *chain.HeadEvent)
	}
	return c.events
}

func (c *LotusChainNotifier) HeadEvents() <-chan *chain.HeadEvent {
	c.mu.Lock()
	ev := c.eventsCh()
	c.mu.Unlock()
	return ev
}

func (c *LotusChainNotifier) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

func (c *LotusChainNotifier) Cancel(err error) {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}
	c.err = err

	// ensure channel is closed even if it was never previously initialised
	if c.events == nil {
		c.events = make(chan *chain.HeadEvent)
	}
	close(c.events)
	c.mu.Unlock()
}

// Run subscribes to ChainNotify and blocks until the context is done or
// an error occurs.
func (c *LotusChainNotifier) Run(ctx context.Context) error {
	node, closer, err := c.opener.Open(ctx)
	if err != nil {
		return xerrors.Errorf("open lens: %w", err)
	}
	defer closer()

	hc, err := node.ChainNotify(ctx)
	if err != nil {
		return xerrors.Errorf("chain notify: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			c.Cancel(ctx.Err())
			return nil
		case headEvents, ok := <-hc:
			if !ok {
				c.Cancel(xerrors.Errorf("ChainNotify channel closed"))
				return nil
			}

			for _, ch := range headEvents {
				stats.Record(ctx, metrics.WatchHeight.M(int64(ch.Val.Height())))
				he := &chain.HeadEvent{
					TipSet: ch.Val,
				}
				switch ch.Type {
				case store.HCCurrent:
					he.Type = chain.HeadEventCurrent
				case store.HCApply:
					he.Type = chain.HeadEventApply
				case store.HCRevert:
					he.Type = chain.HeadEventRevert
				}

				c.events <- he
			}

		}
	}
}

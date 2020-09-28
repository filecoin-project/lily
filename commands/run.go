package commands

import (
	"fmt"
	"math"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/schedule"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/tasks/indexer"
	"github.com/filecoin-project/sentinel-visor/tasks/message"
)

var Run = &cli.Command{
	Name:  "run",
	Usage: "Index and process blocks from the filecoin blockchain",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "max-batch",
			Value: 10,
		},
	},
	Action: func(cctx *cli.Context) error {
		if err := setupLogging(cctx); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		tcloser, err := setupTracing(cctx)
		if err != nil {
			return xerrors.Errorf("setup tracing: %w", err)
		}
		defer tcloser()

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return xerrors.Errorf("setup storage and api: %w", err)
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(ctx); err != nil {
				log.Errorw("close database", "error", err)
			}
		}()

		scheduler := schedule.NewScheduler()

		// Add one indexing task to follow the chain head
		// TODO: enable/disable this with CLI flag
		scheduler.Add(schedule.TaskConfig{
			Name:                "ChainHeadIndexer",
			Task:                indexer.NewChainHeadIndexer(rctx.db, rctx.api),
			Locker:              NewGlobalSingleton(ChainHeadIndexerLockID, rctx.db), // only want one forward indexer anywhere to be running
			RestartOnFailure:    true,
			RestartOnCompletion: true, // we always want the indexer to be running
		})

		// Add one indexing task to walk the chain history
		// TODO: enable/disable this with CLI flag
		scheduler.Add(schedule.TaskConfig{
			Name:                "ChainHistoryIndexer",
			Task:                indexer.NewChainHistoryIndexer(rctx.db, rctx.api),
			Locker:              NewGlobalSingleton(ChainHistoryIndexerLockID, rctx.db), // only want one history indexer anywhere to be running
			RestartOnFailure:    true,
			RestartOnCompletion: false, // run once only
		})

		// TODO: get these from CLI flags
		defaultLeaseTime := time.Minute * 15
		defaultBatchSize := 10
		defaultStateChangeProcessors := 5
		defaultStateChangeMaxHeight := int64(math.MaxInt64)

		// Add several state change tasks to read which actors changed state in each indexed tipset
		for i := 0; i < defaultStateChangeProcessors; i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("ActorStateChangeProcessor%03d", i),
				Task:                actorstate.NewActorStateChangeProcessor(rctx.db, rctx.api, defaultLeaseTime, defaultBatchSize, defaultStateChangeMaxHeight),
				RestartOnFailure:    true,
				RestartOnCompletion: true,
			})
		}

		// TODO: get these from CLI flags
		defaultStateProcessors := 15

		// By default we proces all supported actor types but we could limit here
		defaultActorCodesToProcess := actorstate.SupportedActorCodes()
		defaultActorCodesMaxHeight := int64(math.MaxInt64)

		// Add several state tasks to read actor state from each indexed block
		for i := 0; i < defaultStateProcessors; i++ {
			p, err := actorstate.NewActorStateProcessor(rctx.db, rctx.api, defaultLeaseTime, defaultBatchSize, defaultActorCodesMaxHeight, defaultActorCodesToProcess)
			if err != nil {
				return err
			}
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("ActorStateProcessor%03d", i),
				Task:                p,
				RestartOnFailure:    true,
				RestartOnCompletion: true,
			})
		}

		// TODO: get these from CLI flags
		defaultMessageProcessors := 5
		defaultMessageMaxHeight := int64(math.MaxInt64)

		// Add several message tasks to read messages from indexed tipsets
		for i := 0; i < defaultMessageProcessors; i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("MessageProcessor%03d", i),
				Task:                message.NewMessageProcessor(rctx.db, rctx.api, defaultLeaseTime, defaultBatchSize, defaultMessageMaxHeight),
				RestartOnFailure:    true,
				RestartOnCompletion: true,
			})
		}

		// Start the scheduler and wait for it to complete or to be cancelled.
		return scheduler.Run(ctx)
	},
}

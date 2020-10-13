package commands

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/schedule"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/tasks/indexer"
	"github.com/filecoin-project/sentinel-visor/tasks/message"
	"github.com/filecoin-project/sentinel-visor/tasks/views"
)

var Run = &cli.Command{
	Name:  "run",
	Usage: "Index and process blocks from the filecoin blockchain",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "no-default-tasks",
			Usage:   "When set this flag sets the default number of each task to zero. Tasks will not start unless their relevant CLI flag has been set explicitly.",
			EnvVars: []string{"NO_DEFAULT_TASKS"},
		},
		&cli.Int64Flag{
			Name:    "from",
			Usage:   "Limit actor and message processing to tipsets at or above `HEIGHT`",
			EnvVars: []string{"VISOR_HEIGHT_FROM"},
		},
		&cli.Int64Flag{
			Name:        "to",
			Usage:       "Limit actor and message processing to tipsets at or below `HEIGHT`",
			Value:       math.MaxInt64,
			DefaultText: "MaxInt64",
			EnvVars:     []string{"VISOR_HEIGHT_TO"},
		},
		&cli.BoolFlag{
			Name:    "indexhead",
			Usage:   "Start indexing tipsets by following the chain head",
			Value:   true,
			EnvVars: []string{"VISOR_INDEXHEAD"},
		},
		&cli.IntFlag{
			Name:    "indexhead-confidence",
			Usage:   "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database",
			Value:   25,
			EnvVars: []string{"VISOR_INDEXHEAD_CONFIDENCE"},
		},
		&cli.BoolFlag{
			Name:    "indexhistory",
			Value:   true,
			Usage:   "Start indexing tipsets by walking the chain history",
			EnvVars: []string{"VISOR_INDEXHISTORY"},
		},

		&cli.DurationFlag{
			Name:    "statechange-lease",
			Aliases: []string{"scl"},
			Value:   time.Minute * 15,
			Usage:   "Lease time for the actor state change processor",
			EnvVars: []string{"VISOR_STATECHANGE_LEASE"},
		},
		&cli.IntFlag{
			Name:    "statechange-batch",
			Aliases: []string{"scb"},
			Value:   10,
			Usage:   "Batch size for the actor state change processor",
			EnvVars: []string{"VISOR_STATECHANGE_BATCH"},
		},
		&cli.IntFlag{
			Name:    "statechange-workers",
			Aliases: []string{"scw"},
			Value:   15,
			Usage:   "Number of actor state change processors to start",
			EnvVars: []string{"VISOR_STATECHANGE_WORKERS"},
		},

		&cli.DurationFlag{
			Name:    "actorstate-lease",
			Aliases: []string{"asl"},
			Value:   time.Minute * 15,
			Usage:   "Lease time for the actor state processor",
			EnvVars: []string{"VISOR_ACTORSTATE_LEASE"},
		},
		&cli.IntFlag{
			Name:    "actorstate-batch",
			Aliases: []string{"asb"},
			Value:   10,
			Usage:   "Batch size for the actor state processor",
			EnvVars: []string{"VISOR_ACTORSTATE_BATCH"},
		},
		&cli.IntFlag{
			Name:    "actorstate-workers",
			Aliases: []string{"asw"},
			Value:   15,
			Usage:   "Number of actor state processors to start",
			EnvVars: []string{"VISOR_ACTORSTATE_WORKERS"},
		},
		&cli.StringSliceFlag{
			Name:        "actorstate-include",
			Usage:       "List of actor codes that should be procesed by actor state processors",
			DefaultText: "all supported",
			EnvVars:     []string{"VISOR_ACTORSTATE_INCLUDE"},
		},
		&cli.StringSliceFlag{
			Name:        "actorstate-exclude",
			Usage:       "List of actor codes that should be not be procesed by actor state processors",
			DefaultText: "none",
			EnvVars:     []string{"VISOR_ACTORSTATE_EXCLUDE"},
		},

		&cli.DurationFlag{
			Name:    "message-lease",
			Aliases: []string{"ml"},
			Value:   time.Minute * 15,
			Usage:   "Lease time for the message processor",
			EnvVars: []string{"VISOR_MESSAGE_LEASE"},
		},
		&cli.IntFlag{
			Name:    "message-batch",
			Aliases: []string{"mb"},
			Value:   10,
			Usage:   "Batch size for the message processor",
			EnvVars: []string{"VISOR_MESSAGE_BATCH"},
		},
		&cli.IntFlag{
			Name:    "message-workers",
			Aliases: []string{"mw"},
			Value:   15,
			Usage:   "Number of message processors to start",
			EnvVars: []string{"VISOR_MESSAGE_WORKERS"},
		},

		&cli.DurationFlag{
			Name:    "gasoutputs-lease",
			Aliases: []string{"gol"},
			Value:   time.Minute * 15,
			Usage:   "Lease time for the gas outputs processor",
			EnvVars: []string{"VISOR_GASOUTPUTS_LEASE"},
		},
		&cli.IntFlag{
			Name:    "gasoutputs-batch",
			Aliases: []string{"gob"},
			Value:   500, // can be high because we don't hit the lotus api
			Usage:   "Batch size for the gas outputs processor",
			EnvVars: []string{"VISOR_GASOUTPUTS_BATCH"},
		},
		&cli.IntFlag{
			Name:    "gasoutputs-workers",
			Aliases: []string{"gow"},
			Value:   15,
			Usage:   "Number of gas outputs processors to start",
			EnvVars: []string{"VISOR_GASOUTPUTS_WORKERS"},
		},

		&cli.DurationFlag{
			Name:    "chainvis-refresh-rate",
			Aliases: []string{"crr"},
			Value:   0,
			Usage:   "Refresh frequency for chain visualization views (0 = disables refresh)",
			EnvVars: []string{"VISOR_CHAINVIS_REFRESH"},
		},
	},
	Action: func(cctx *cli.Context) error {
		// Validate flags
		heightFrom := cctx.Int64("from")
		heightTo := cctx.Int64("to")
		actorCodes, err := getActorCodes(cctx)
		if err != nil {
			return err
		}

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
		if boolFlag(cctx, "indexhead") {
			scheduler.Add(schedule.TaskConfig{
				Name:                "ChainHeadIndexer",
				Task:                indexer.NewChainHeadIndexer(rctx.db, rctx.api, cctx.Int("indexhead-confidence")),
				Locker:              NewGlobalSingleton(ChainHeadIndexerLockID, rctx.db), // only want one forward indexer anywhere to be running
				RestartOnFailure:    true,
				RestartOnCompletion: true, // we always want the indexer to be running
				RestartDelay:        time.Minute,
			})
		}

		// Add one indexing task to walk the chain history
		if boolFlag(cctx, "indexhistory") {
			scheduler.Add(schedule.TaskConfig{
				Name:                "ChainHistoryIndexer",
				Task:                indexer.NewChainHistoryIndexer(rctx.db, rctx.api),
				Locker:              NewGlobalSingleton(ChainHistoryIndexerLockID, rctx.db), // only want one history indexer anywhere to be running
				RestartOnFailure:    true,
				RestartOnCompletion: true,
				RestartDelay:        time.Minute,
			})
		}

		// Add several state change tasks to read which actors changed state in each indexed tipset
		for i := 0; i < intFlag(cctx, "statechange-workers"); i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("ActorStateChangeProcessor%03d", i),
				Task:                actorstate.NewActorStateChangeProcessor(rctx.db, rctx.api, cctx.Duration("statechange-lease"), cctx.Int("statechange-batch"), heightFrom, heightTo),
				RestartOnFailure:    true,
				RestartOnCompletion: true,
			})
		}

		// Add several state tasks to read actor state from each indexed block
		for i := 0; i < intFlag(cctx, "actorstate-workers"); i++ {
			p, err := actorstate.NewActorStateProcessor(rctx.db, rctx.api, cctx.Duration("actorstate-lease"), cctx.Int("actorstate-batch"), heightFrom, heightTo, actorCodes)
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

		// Add several message tasks to read messages from indexed tipsets
		for i := 0; i < intFlag(cctx, "message-workers"); i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("MessageProcessor%03d", i),
				Task:                message.NewMessageProcessor(rctx.db, rctx.api, cctx.Duration("message-lease"), cctx.Int("message-batch"), heightFrom, heightTo),
				RestartOnFailure:    true,
				RestartOnCompletion: true,
			})
		}

		// Add several gas output tasks to read gas outputs from indexed messages
		for i := 0; i < intFlag(cctx, "gasoutputs-workers"); i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("GasOutputsProcessor%03d", i),
				Task:                message.NewGasOutputsProcessor(rctx.db, rctx.api, cctx.Duration("gasoutputs-lease"), cctx.Int("gasoutputs-batch"), heightFrom, heightTo),
				RestartOnFailure:    true,
				RestartOnCompletion: true,
			})
		}

		// Include optional refresher for Chain Visualization views
		// Zero duration will cause ChainVisRefresher to exit and should not restart
		scheduler.Add(schedule.TaskConfig{
			Name:                "ChainVisRefresher",
			Locker:              NewGlobalSingleton(ChainVisRefresherLockID, rctx.db), // only need one chain vis refresher anywhere
			Task:                views.NewChainVisRefresher(rctx.db, cctx.Duration("chainvis-refresh-rate")),
			RestartOnFailure:    true,
			RestartOnCompletion: false,
		})

		// Start the scheduler and wait for it to complete or to be cancelled.
		err = scheduler.Run(ctx)
		if !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	},
}

// getActorCodes parses the cli flags to obtain a list of actor codes for the actor state processor. We support some
// common short names for actors or the cid of the actor code.
func getActorCodes(cctx *cli.Context) ([]cid.Cid, error) {
	include := cctx.StringSlice("actorstate-include")
	exclude := cctx.StringSlice("actorstate-exclude")
	if len(include) == 0 && len(exclude) == 0 {
		// By default we process all supported actor types
		return actorstate.SupportedActorCodes(), nil
	}

	if len(include) != 0 && len(exclude) != 0 {
		return nil, fmt.Errorf("cannot specify both actorstate-include and actorstate-exclude")
	}

	if len(include) != 0 {
		return parseActorCodes(include)
	}

	excludeCids, err := parseActorCodes(include)
	if err != nil {
		return nil, err
	}

	var codes []cid.Cid
	for _, c := range actorstate.SupportedActorCodes() {
		excluded := false
		for _, ex := range excludeCids {
			if c.Equals(ex) {
				// exclude it
				excluded = true
				break
			}
		}
		if !excluded {
			codes = append(codes, c)
		}
	}

	if len(codes) == 0 {
		return nil, fmt.Errorf("all supported actors have been excluded")
	}

	return codes, nil
}

func parseActorCodes(ss []string) ([]cid.Cid, error) {
	var codes []cid.Cid
	for _, s := range ss {
		c, ok := actorNamesToCodes[s]
		if ok {
			codes = append(codes, c)
			continue
		}

		var err error
		c, err = cid.Decode(s)
		if err != nil {
			return nil, fmt.Errorf("invalid cid: %w", err)
		}
		codes = append(codes, c)
	}

	return codes, nil
}

var actorNamesToCodes = map[string]cid.Cid{
	"fil/2/system":           builtin.SystemActorCodeID,
	"fil/2/init":             builtin.InitActorCodeID,
	"fil/2/cron":             builtin.CronActorCodeID,
	"fil/2/storagepower":     builtin.StoragePowerActorCodeID,
	"fil/2/storageminer":     builtin.StorageMinerActorCodeID,
	"fil/2/storagemarket":    builtin.StorageMarketActorCodeID,
	"fil/2/paymentchannel":   builtin.PaymentChannelActorCodeID,
	"fil/2/reward":           builtin.RewardActorCodeID,
	"fil/2/verifiedregistry": builtin.VerifiedRegistryActorCodeID,
	"fil/2/account":          builtin.AccountActorCodeID,
	"fil/2/multisig":         builtin.MultisigActorCodeID,
}

// boolFlag always returns the value of a boolean flag if set. If not set
// then the default value will be returned unless the --no-default-tasks is
// specified. In that case false is returned.
func boolFlag(cctx *cli.Context, flagname string) bool {
	if cctx.Bool("no-default-tasks") && !cctx.IsSet(flagname) {
		return false
	}
	return cctx.Bool(flagname)
}

// intFlag always returns the value of an int flag if set. If not set
// then the default value will be returned unless the --no-default-tasks is
// specified. In that case 0 is returned.
func intFlag(cctx *cli.Context, flagname string) int {
	if cctx.Bool("no-default-tasks") && !cctx.IsSet(flagname) {
		return 0
	}
	return cctx.Int(flagname)
}

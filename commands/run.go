package commands

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/schedule"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/tasks/chain"
	"github.com/filecoin-project/sentinel-visor/tasks/indexer"
	"github.com/filecoin-project/sentinel-visor/tasks/message"
	"github.com/filecoin-project/sentinel-visor/tasks/stats"
	"github.com/filecoin-project/sentinel-visor/tasks/views"
	"github.com/filecoin-project/sentinel-visor/version"
)

var Run = &cli.Command{
	Name:  "run",
	Usage: "Index and process blocks from the filecoin blockchain",
	Flags: []cli.Flag{
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
			Value:   false,
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
			Value:   false,
			Usage:   "Start indexing tipsets by walking the chain history",
			EnvVars: []string{"VISOR_INDEXHISTORY"},
		},
		&cli.IntFlag{
			Name:    "indexhistory-batch",
			Aliases: []string{"ihb"},
			Value:   25,
			Usage:   "Batch size for the chain history indexer",
			EnvVars: []string{"VISOR_INDEXHISTORY_BATCH"},
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
			Value:   0,
			Usage:   "Number of actor state change processors to start",
			EnvVars: []string{"VISOR_STATECHANGE_WORKERS"},
		},

		&cli.DurationFlag{
			Name:    "actorstate-lease",
			Aliases: []string{"asl"},
			Value:   time.Minute * 15,
			Usage:   "Lease time for the actor state processor. Set to zero to avoid using leases.",
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
			Value:   0,
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
			Value:   0,
			Usage:   "Number of message processors to start",
			EnvVars: []string{"VISOR_MESSAGE_WORKERS"},
		},
		&cli.BoolFlag{
			Name:    "derive-parsed-messages",
			Aliases: []string{"dpm"},
			Value:   false,
			Usage:   "Fill the parsed messages table when processing messages",
			EnvVars: []string{"VISOR_MESSAGE_PARSED"},
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
			Value:   0,
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

		&cli.DurationFlag{
			Name:    "processingstats-refresh-rate",
			Aliases: []string{"psr"},
			Value:   0,
			Usage:   "Refresh frequency for processing stats (0 = disables refresh)",
			EnvVars: []string{"VISOR_PROCESSINGSTATS_REFRESH"},
		},

		&cli.IntFlag{
			Name:    "chaineconomics-workers",
			Aliases: []string{"cew"},
			Value:   0,
			Usage:   "Number of chain economics processors to start",
			EnvVars: []string{"VISOR_CHAINECONOMICS_WORKERS"},
		},
		&cli.IntFlag{
			Name:    "chaineconomics-batch",
			Aliases: []string{"ceb"},
			Value:   50, // chain economics processing is quite fast
			Usage:   "Batch size for the chain economics processor",
			EnvVars: []string{"VISOR_CHAINECONOMICS_BATCH"},
		},
		&cli.DurationFlag{
			Name:    "chaineconomics-lease",
			Aliases: []string{"cel"},
			Value:   time.Minute * 15,
			Usage:   "Lease time for the chain economics processor",
			EnvVars: []string{"VISOR_CHAINECONOMICS_LEASE"},
		},

		&cli.DurationFlag{
			Name:    "task-delay",
			Aliases: []string{"td"},
			Value:   500 * time.Millisecond,
			Usage:   "Base time to wait between starting tasks (jitter is added)",
			EnvVars: []string{"VISOR_TASK_DELAY"},
		},
	},
	Action: func(cctx *cli.Context) error {
		// Validate flags
		heightFrom := cctx.Int64("from")
		heightTo := cctx.Int64("to")

		if heightFrom > heightTo {
			return xerrors.Errorf("--from must not be greater than --to")
		}

		actorCodes, err := getActorCodes(cctx)
		if err != nil {
			return err
		}

		if err := setupLogging(cctx); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		log.Infof("Visor version:%s", version.String())

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

		scheduler := schedule.NewScheduler(cctx.Duration("task-delay"))

		// Add one indexing task to follow the chain head
		if cctx.Bool("indexhead") {
			scheduler.Add(schedule.TaskConfig{
				Name:                "ChainHeadIndexer",
				Task:                indexer.NewChainHeadIndexer(rctx.db, rctx.opener, cctx.Int("indexhead-confidence")),
				Locker:              NewGlobalSingleton(ChainHeadIndexerLockID, rctx.db), // only want one forward indexer anywhere to be running
				ExitOnFailure:       true,
				RestartOnFailure:    false,
				RestartOnCompletion: false,
				RestartDelay:        time.Minute,
			})
		}

		// Add one indexing task to walk the chain history
		if cctx.Bool("indexhistory") {
			scheduler.Add(schedule.TaskConfig{
				Name:                "ChainHistoryIndexer",
				Task:                indexer.NewChainHistoryIndexer(rctx.db, rctx.opener, cctx.Int("indexhistory-batch")),
				Locker:              NewGlobalSingleton(ChainHistoryIndexerLockID, rctx.db), // only want one history indexer anywhere to be running
				ExitOnFailure:       true,
				RestartOnFailure:    false,
				RestartOnCompletion: false,
				RestartDelay:        time.Minute,
			})
		}

		// Add several state change tasks to read which actors changed state in each indexed tipset
		for i := 0; i < cctx.Int("statechange-workers"); i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("ActorStateChangeProcessor%03d", i),
				Task:                actorstate.NewActorStateChangeProcessor(rctx.db, rctx.opener, cctx.Duration("statechange-lease"), cctx.Int("statechange-batch"), heightFrom, heightTo),
				ExitOnFailure:       true,
				RestartOnFailure:    false,
				RestartOnCompletion: false,
			})
		}

		// Add several state tasks to read actor state from each indexed block
		// actor state processing cannot include genesis
		actorStateHeightFrom := heightFrom
		if actorStateHeightFrom == 0 {
			actorStateHeightFrom = 1
		}

		// If we are not using leases then further subdivide work by height to avoid workers processing the same actor states
		if cctx.Duration("actorstate-lease") == 0 {
			if cctx.Int("actorstate-workers") > 1 && heightTo > estimateCurrentEpoch()*2 {
				log.Warnf("--to is set to an unexpectedly high epoch which will likely result in some workers not being assigned a useful height range")
			}

			hr := heightRange{min: actorStateHeightFrom, max: heightTo}
			srs := hr.divide(cctx.Int("actorstate-workers"))
			for i, sr := range srs {
				p, err := actorstate.NewActorStateProcessor(rctx.db, rctx.opener, 0, cctx.Int("actorstate-batch"), sr.min, sr.max, actorCodes, false)
				if err != nil {
					return err
				}
				log.Debugf("scheduling actor state processor with height range %d to %d", sr.min, sr.max)
				scheduler.Add(schedule.TaskConfig{
					Name:                fmt.Sprintf("ActorStateProcessor%03d", i),
					Task:                p,
					ExitOnFailure:       true,
					RestartOnFailure:    false,
					RestartOnCompletion: false,
					RestartDelay:        time.Minute,
				})
			}
		} else {
			// Use workers with leasing
			for i := 0; i < cctx.Int("actorstate-workers"); i++ {
				p, err := actorstate.NewActorStateProcessor(rctx.db, rctx.opener, cctx.Duration("actorstate-lease"), cctx.Int("actorstate-batch"), actorStateHeightFrom, heightTo, actorCodes, true)
				if err != nil {
					return err
				}
				scheduler.Add(schedule.TaskConfig{
					Name:                fmt.Sprintf("ActorStateProcessor%03d", i),
					Task:                p,
					ExitOnFailure:       true,
					RestartOnFailure:    false,
					RestartOnCompletion: false,
					RestartDelay:        time.Minute,
				})
			}
		}
		// Add several message tasks to read messages from indexed tipsets
		for i := 0; i < cctx.Int("message-workers"); i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("MessageProcessor%03d", i),
				Task:                message.NewMessageProcessor(rctx.db, rctx.opener, cctx.Duration("message-lease"), cctx.Int("message-batch"), cctx.Bool("derive-parsed-messages"), heightFrom, heightTo),
				ExitOnFailure:       true,
				RestartOnFailure:    false,
				RestartOnCompletion: false,
				RestartDelay:        time.Minute,
			})
		}

		// If we are not using leases then further subdivide work by height to avoid workers processing the same actor states
		if cctx.Duration("gasoutputs-lease") == 0 {
			if cctx.Int("gasoutputs-workers") > 1 && heightTo > estimateCurrentEpoch()*2 {
				log.Warnf("--to is set to an unexpectedly high epoch which will likely result in some workers not being assigned a useful height range")
			}

			hr := heightRange{min: heightFrom, max: heightTo}
			srs := hr.divide(cctx.Int("gasoutputs-workers"))
			for i, sr := range srs {
				log.Debugf("scheduling gas outputs state processor with height range %d to %d", sr.min, sr.max)
				scheduler.Add(schedule.TaskConfig{
					Name:                fmt.Sprintf("GasOutputsProcessor%03d", i),
					Task:                message.NewGasOutputsProcessor(rctx.db, rctx.opener, cctx.Duration("gasoutputs-lease"), cctx.Int("gasoutputs-batch"), sr.min, sr.max, false),
					RestartOnFailure:    true,
					RestartOnCompletion: true,
					RestartDelay:        time.Minute,
				})
			}
		} else {
			// Add several gas output tasks to read gas outputs from indexed messages
			for i := 0; i < cctx.Int("gasoutputs-workers"); i++ {
				scheduler.Add(schedule.TaskConfig{
					Name:                fmt.Sprintf("GasOutputsProcessor%03d", i),
					Task:                message.NewGasOutputsProcessor(rctx.db, rctx.opener, cctx.Duration("gasoutputs-lease"), cctx.Int("gasoutputs-batch"), heightFrom, heightTo, true),
					RestartOnFailure:    true,
					RestartOnCompletion: true,
					RestartDelay:        time.Minute,
				})
			}
		}

		// Add several chain economics tasks to read gas outputs from indexed messages
		for i := 0; i < cctx.Int("chaineconomics-workers"); i++ {
			scheduler.Add(schedule.TaskConfig{
				Name:                fmt.Sprintf("ChainEconomicsProcessor%03d", i),
				Task:                chain.NewChainEconomicsProcessor(rctx.db, rctx.opener, cctx.Duration("chaineconomics-lease"), cctx.Int("chaineconomics-batch"), heightFrom, heightTo),
				ExitOnFailure:       true,
				RestartOnFailure:    false,
				RestartOnCompletion: false,
				RestartDelay:        time.Minute,
			})
		}

		// Include optional refresher for Chain Visualization views
		// Zero duration will cause ChainVisRefresher to exit and should not restart
		if cctx.Duration("chainvis-refresh-rate") != 0 {
			scheduler.Add(schedule.TaskConfig{
				Name:                "ChainVisRefresher",
				Locker:              NewGlobalSingleton(ChainVisRefresherLockID, rctx.db), // only need one chain vis refresher anywhere
				Task:                views.NewChainVisRefresher(rctx.db, cctx.Duration("chainvis-refresh-rate")),
				ExitOnFailure:       true,
				RestartOnFailure:    false,
				RestartOnCompletion: false,
				RestartDelay:        time.Minute,
			})
		}
		// Include optional refresher for processing stats
		if cctx.Duration("processingstats-refresh-rate") != 0 {
			scheduler.Add(schedule.TaskConfig{
				Name:                "ProcessingStatsRefresher",
				Locker:              NewGlobalSingleton(ProcessingStatsRefresherLockID, rctx.db),
				Task:                stats.NewProcessingStatsRefresher(rctx.db, cctx.Duration("processingstats-refresh-rate")),
				RestartOnFailure:    true,
				RestartOnCompletion: true,
				RestartDelay:        time.Minute,
			})
		}

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
	"fil/1/system":           builtin.SystemActorCodeID,
	"fil/1/init":             builtin.InitActorCodeID,
	"fil/1/cron":             builtin.CronActorCodeID,
	"fil/1/storagepower":     builtin.StoragePowerActorCodeID,
	"fil/1/storageminer":     builtin.StorageMinerActorCodeID,
	"fil/1/storagemarket":    builtin.StorageMarketActorCodeID,
	"fil/1/paymentchannel":   builtin.PaymentChannelActorCodeID,
	"fil/1/reward":           builtin.RewardActorCodeID,
	"fil/1/verifiedregistry": builtin.VerifiedRegistryActorCodeID,
	"fil/1/account":          builtin.AccountActorCodeID,
	"fil/1/multisig":         builtin.MultisigActorCodeID,
	"fil/2/system":           builtin2.SystemActorCodeID,
	"fil/2/init":             builtin2.InitActorCodeID,
	"fil/2/cron":             builtin2.CronActorCodeID,
	"fil/2/storagepower":     builtin2.StoragePowerActorCodeID,
	"fil/2/storageminer":     builtin2.StorageMinerActorCodeID,
	"fil/2/storagemarket":    builtin2.StorageMarketActorCodeID,
	"fil/2/paymentchannel":   builtin2.PaymentChannelActorCodeID,
	"fil/2/reward":           builtin2.RewardActorCodeID,
	"fil/2/verifiedregistry": builtin2.VerifiedRegistryActorCodeID,
	"fil/2/account":          builtin2.AccountActorCodeID,
	"fil/2/multisig":         builtin2.MultisigActorCodeID,
}

type heightRange struct {
	min, max int64
}

// divide divides a range into n equal subranges. The last range may be undersized.
func (h heightRange) divide(n int) []heightRange {
	if n <= 1 {
		return []heightRange{h}
	}
	size := h.max - h.min
	if size != math.MaxInt64 {
		size++ // +1 since the range is inclusive
	}
	if int64(n) > size {
		panic(fmt.Sprintf("can't subdivide a range of %d into %d pieces", size, n))
	}
	step := size/int64(n) - 1

	subs := make([]heightRange, n)
	subs[0].min = h.min
	subs[0].max = h.min + step

	for i := 1; i < n; i++ {
		subs[i].min = subs[i-1].max + 1
		subs[i].max = subs[i].min + step
	}

	subs[n-1].max = h.max
	return subs
}

var mainnetGenesis = time.Date(2020, 8, 24, 22, 0, 0, 0, time.UTC)

func estimateCurrentEpoch() int64 {
	return int64(time.Since(mainnetGenesis) / (builtin.EpochDurationSeconds))
}

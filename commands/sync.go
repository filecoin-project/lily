package commands

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	lotuscli "github.com/filecoin-project/lotus/cli"
	cid "github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/chain/actors/builtin"

	"github.com/filecoin-project/lily/lens/lily"
)

var SyncCmd = &cli.Command{
	Name:  "sync",
	Usage: "Inspect or interact with the chain syncer",
	Subcommands: []*cli.Command{
		SyncStatusCmd,
		SyncWaitCmd,
	},
}

var SyncStatusCmd = &cli.Command{
	Name:  "status",
	Usage: "Report sync status of a running lily daemon",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		state, err := lapi.SyncState(ctx)
		if err != nil {
			return err
		}

		fmt.Println("sync status:")
		for _, ss := range state.ActiveSyncs {
			fmt.Printf("worker %d:\n", ss.WorkerID)
			var base, target []cid.Cid
			var heightDiff int64
			var theight abi.ChainEpoch
			if ss.Base != nil {
				base = ss.Base.Cids()
				heightDiff = int64(ss.Base.Height())
			}
			if ss.Target != nil {
				target = ss.Target.Cids()
				heightDiff = int64(ss.Target.Height()) - heightDiff
				theight = ss.Target.Height()
			} else {
				heightDiff = 0
			}
			fmt.Printf("\tBase:\t%s\n", base)
			fmt.Printf("\tTarget:\t%s (%d)\n", target, theight)
			fmt.Printf("\tHeight diff:\t%d\n", heightDiff)
			fmt.Printf("\tStage: %s\n", ss.Stage)
			fmt.Printf("\tHeight: %d\n", ss.Height)
			if ss.End.IsZero() {
				if !ss.Start.IsZero() {
					fmt.Printf("\tElapsed: %s\n", time.Since(ss.Start))
				}
			} else {
				fmt.Printf("\tElapsed: %s\n", ss.End.Sub(ss.Start))
			}
			if ss.Stage == api.StageSyncErrored {
				fmt.Printf("\tError: %s\n", ss.Message)
			}
		}
		return nil
	},
}

type syncWaitOptions struct{
	watch bool
	untilEpoch int
}

var opts syncWaitOptions

var SyncWaitCmd = &cli.Command{
	Name:  "wait",
	Usage: "Wait for sync to be complete",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "watch",
			Usage: "don't exit after node is synced",
			Destination: &opts.watch,
		},
		&cli.IntFlag{
			Name:  "until-epoch",
			Usage: "returns after the epoch provided",
			Destination: &opts.untilEpoch,
		},
	},
	Action: func(cctx *cli.Context) error {
		if opts.untilEpoch > 0 && opts.watch {
			return fmt.Errorf("until-epoch and watch flags cannot be used together")
		}
		// ensure default doesn't cause an internal exit by making the value
		// impossibly large
		if opts.untilEpoch == 0 {
			opts.untilEpoch = abi.ChainEpoch(math.MaxInt64)
		}

		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		return SyncWait(ctx, lapi, opts)
	},
}

func SyncWait(ctx context.Context, lapi lily.LilyAPI, o syncWaitOptions) error {
	tick := time.Second / 4

	lastLines := 0
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	samples := 8
	i := 0
	var firstApp, app, lastApp uint64

	state, err := lapi.SyncState(ctx)
	if err != nil {
		return err
	}
	firstApp = state.VMApplied

	for {
		state, err := lapi.SyncState(ctx)
		if err != nil {
			return err
		}

		if len(state.ActiveSyncs) == 0 {
			time.Sleep(time.Second)
			continue
		}

		head, err := lapi.ChainHead(ctx)
		if err != nil {
			return err
		}

		// stopping one epoch after requested to ensure receipts are processed
		// and included in the repo state
		if head.Height() > abi.ChainEpoch(o.untilEpoch + 1) {
			fmt.Println("\nDone!")
			return nil
		}

		working := -1
		for i, ss := range state.ActiveSyncs {
			switch ss.Stage {
			case api.StageSyncComplete:
			case api.StageIdle:
				// not complete, not actively working
			default:
				working = i
			}
		}

		if working == -1 {
			working = len(state.ActiveSyncs) - 1
		}

		ss := state.ActiveSyncs[working]
		workerID := ss.WorkerID

		var baseHeight abi.ChainEpoch
		var target []cid.Cid
		var theight abi.ChainEpoch
		var heightDiff int64

		if ss.Base != nil {
			baseHeight = ss.Base.Height()
			heightDiff = int64(ss.Base.Height())
		}
		if ss.Target != nil {
			target = ss.Target.Cids()
			theight = ss.Target.Height()
			heightDiff = int64(ss.Target.Height()) - heightDiff
		} else {
			heightDiff = 0
		}

		for i := 0; i < lastLines; i++ {
			fmt.Print("\r\x1b[2K\x1b[A")
		}

		fmt.Printf("Worker: %d; Base: %d; Target: %d (diff: %d)\n", workerID, baseHeight, theight, heightDiff)
		fmt.Printf("State: %s; Current Epoch: %d; Todo: %d\n", ss.Stage, ss.Height, theight-ss.Height)
		lastLines = 2

		if i%samples == 0 {
			lastApp = app
			app = state.VMApplied - firstApp
		}
		if i > 0 {
			fmt.Printf("Validated %d messages (%d per second)\n", state.VMApplied-firstApp, (app-lastApp)*uint64(time.Second/tick)/uint64(samples))
			lastLines++
		}

		_ = target // todo: maybe print? (creates a bunch of line wrapping issues with most tipsets)

		if !o.watch && time.Now().Unix()-int64(head.MinTimestamp()) < int64(builtin.EpochDurationSeconds) {
			fmt.Println("\nDone!")
			return nil
		}

		select {
		case <-ctx.Done():
			fmt.Println("\nExit by user")
			return nil
		case <-ticker.C:
		}

		i++
	}
}

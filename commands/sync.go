package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	cid "github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/blocks"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	lotuscli "github.com/filecoin-project/lotus/cli"
)

type SyncStatus struct {
	Stage  api.SyncStateStage
	Height abi.ChainEpoch
}

var SyncCmd = &cli.Command{
	Name:  "sync",
	Usage: "Inspect or interact with the chain syncer",
	Subcommands: []*cli.Command{
		SyncStatusCmd,
		SyncWaitCmd,
		SyncIncomingBlockCmd,
	},
}

var SyncStatusCmd = &cli.Command{
	Name:  "status",
	Usage: "Report sync status of a running lily daemon",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "output",
			Usage:    "Print only the current sync stage at the latest height. One of [text, json]",
			Aliases:  []string{"o"},
			Required: false,
		},
	},
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

		output := cctx.String("output")

		var max abi.ChainEpoch = -1
		maxStateSync := api.StageIdle
		for _, ss := range state.ActiveSyncs {
			if max < ss.Height && maxStateSync <= ss.Stage {
				max = ss.Height
				maxStateSync = ss.Stage
			}

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

			switch output {
			case "json":
				j, err := json.Marshal(SyncStatus{Stage: maxStateSync, Height: max})
				if err != nil {
					return err
				}
				fmt.Printf(string(j) + "\n")
			case "":
				fmt.Printf("worker %d:\n", ss.WorkerID)
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
			case "text":
				fallthrough
			default:
				fmt.Printf("%s %d\n", maxStateSync, max)
			}

			if ss.Stage == api.StageSyncErrored && output != "json" {
				fmt.Printf("\tError: %s\n", ss.Message)
			}

		}
		return nil
	},
}

var SyncWaitCmd = &cli.Command{
	Name:  "wait",
	Usage: "Wait for sync to be complete",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "watch",
			Usage: "don't exit after node is synced",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		return SyncWait(ctx, lapi, cctx.Bool("watch"))
	},
}

func SyncWait(ctx context.Context, lapi lily.LilyAPI, watch bool) error {
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

		if !watch && time.Now().Unix()-int64(head.MinTimestamp()) < int64(builtin.EpochDurationSeconds) {
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

type syncOpts struct {
	config     string
	storage    string
	confidence int
}

var syncFlags syncOpts

type SyncingState struct {
	UnsyncedBlockHeadersByEpoch map[int64][]*blocks.UnsyncedBlockHeader
	MapMutex                    sync.Mutex
	Confidence                  int64
	Storage                     model.Storage
	StorageMutex                sync.Mutex
}

func (ss *SyncingState) SetBlockHeaderToMap(block *blocks.UnsyncedBlockHeader) {
	ss.MapMutex.Lock()
	defer ss.MapMutex.Unlock()
	ss.UnsyncedBlockHeadersByEpoch[block.Height] = append(ss.UnsyncedBlockHeadersByEpoch[block.Height], block)
}

func (ss *SyncingState) PersistBlocks(ctx context.Context, blocks blocks.UnsyncedBlockHeaders) error {
	ss.StorageMutex.Lock()
	defer ss.StorageMutex.Unlock()

	return ss.Storage.PersistBatch(ctx, blocks)
}

var SyncIncomingBlockCmd = &cli.Command{
	Name:  "blocks",
	Usage: "Start to get incoming block",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Specify path of config file to use.",
			EnvVars:     []string{"LILY_CONFIG"},
			Destination: &syncFlags.config,
		},
		&cli.StringFlag{
			Name:        "storage",
			Usage:       "Specify the storage to use, if persisting the displayed output.",
			Destination: &syncFlags.storage,
		},
		&cli.IntFlag{
			Name:        "confidence",
			Usage:       "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database.",
			EnvVars:     []string{"LILY_CONFIDENCE"},
			Value:       2,
			Destination: &syncFlags.confidence,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		// Set up a context that is canceled when the command is interrupted
		ctx, cancel := SetupContextWithCancel(ctx)
		defer cancel()

		strg, err := SetupStorage(syncFlags.config, syncFlags.storage)
		if err != nil {
			return err
		}

		state := &SyncingState{
			UnsyncedBlockHeadersByEpoch: make(map[int64][]*blocks.UnsyncedBlockHeader),
			Confidence:                  int64(syncFlags.confidence),
			Storage:                     strg,
			MapMutex:                    sync.Mutex{},
			StorageMutex:                sync.Mutex{},
		}

		go detectOrphanBlocks(ctx, lapi, state)
		go getIncomingBlocks(ctx, lapi, state)

		<-ctx.Done()
		return nil
	},
}

func detectOrphanBlocks(ctx context.Context, lapi lily.LilyAPI, state *SyncingState) {
	for range time.Tick(30 * time.Second) {
		state.MapMutex.Lock()

		// Get the latestEpoch in map
		latestEpoch := int64(0)
		for k := range state.UnsyncedBlockHeadersByEpoch {
			if k > latestEpoch {
				latestEpoch = k
			}
		}

		// Check old tipset
		targetEpoch := latestEpoch - state.Confidence
		oldEpoches := []int64{}
		for epoch, unsyncedBlocks := range state.UnsyncedBlockHeadersByEpoch {
			if epoch <= targetEpoch {
				// Store the old tipset, we should clear it after checking
				oldEpoches = append(oldEpoches, epoch)

				oldTs, err := lapi.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), types.EmptyTSK)
				if err != nil {
					log.Errorf("Error at getting the old tipset: %v", err)
					continue
				}
				log.Infof("Get header cids: %v at Height: %v", oldTs.Cids(), oldTs.Height())

				// Verify whether the unsynced block exists within the tipset or not.
				cidMap := make(map[string]bool)
				for _, cid := range oldTs.Cids() {
					cidMap[cid.String()] = true
				}
				orphanBlocks := blocks.UnsyncedBlockHeaders{}
				for _, block := range unsyncedBlocks {
					if _, exists := cidMap[block.Cid]; !exists {
						block.IsOrphan = true
						orphanBlocks = append(orphanBlocks, block)
						log.Errorf("Detect orphan block cid: %v at height: %v", block.Cid, block.Height)
					}
				}

				// To do set the orphan to Storage
				if len(orphanBlocks) > 0 && state.Storage != nil {
					err := state.PersistBlocks(ctx, orphanBlocks)
					if err != nil {
						log.Errorf("Error at persisting the orphan blocks: %v", err)
					}
				}
			}
		}

		// Clean the map
		for _, epoch := range oldEpoches {
			delete(state.UnsyncedBlockHeadersByEpoch, epoch)
		}
		state.MapMutex.Unlock()
	}
}

func getIncomingBlocks(ctx context.Context, lapi lily.LilyAPI, state *SyncingState) {
	incomingBlocks, err := lapi.SyncIncomingBlocks(ctx)
	if err != nil {
		log.Error(err)
		return
	}

	for bh := range incomingBlocks {
		block := blocks.NewUnsyncedBlockHeader(bh)
		state.SetBlockHeaderToMap(block)
		if state.Storage == nil {
			log.Infof("Block Height: %v, Miner: %v, Cid: %v", block.Height, block.Miner, block.Cid)
		} else {
			err = state.PersistBlocks(ctx, blocks.UnsyncedBlockHeaders{block})
			if err != nil {
				log.Errorf("Error at persisting the unsynced block headers: %v", err)
			}
		}
	}
}

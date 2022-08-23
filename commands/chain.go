package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	lotusbuild "github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/ipfs/go-cid"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v2"
	"gopkg.in/cheggaaa/pb.v1"

	"github.com/filecoin-project/lily/chain/actors"
	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/lens/util"
)

var ChainCmd = &cli.Command{
	Name:  "chain",
	Usage: "Interact with filecoin blockchain",
	Subcommands: []*cli.Command{
		ChainHeadCmd,
		ChainGetBlock,
		ChainReadObjCmd,
		ChainStatObjCmd,
		ChainGetMsgCmd,
		ChainListCmd,
		ChainSetHeadCmd,
		ChainActorCodesCmd,
		ChainStateInspect,
		ChainStateCompute,
		ChainStateComputeRange,
	},
}

var ChainActorCodesCmd = &cli.Command{
	Name:  "actor-codes",
	Usage: "Print actor codes and names",
	Action: func(cctx *cli.Context) error {
		if err := printSortedActorVersions(init_.VersionCodes()); err != nil {
			return err
		}
		if err := printSortedActorVersions(power.VersionCodes()); err != nil {
			return err
		}
		if err := printSortedActorVersions(miner.VersionCodes()); err != nil {
			return err
		}
		if err := printSortedActorVersions(market.VersionCodes()); err != nil {
			return err
		}
		if err := printSortedActorVersions(verifreg.VersionCodes()); err != nil {
			return err
		}
		if err := printSortedActorVersions(multisig.VersionCodes()); err != nil {
			return err
		}
		if err := printSortedActorVersions(reward.VersionCodes()); err != nil {
			return err
		}
		return nil
	},
}

func marshalReport(reports []*lily.StateReport, verbose bool) ([]byte, error) {
	type stateHeights struct{
		Newest int64 `json:"newest"`
		Oldest int64 `json:"oldest"`
	}
	type summarizedHeights struct {
		Messages stateHeights   `json:"messages"`
		StateRoots stateHeights `json:"stateroots"`
	}
	type hasState struct{
		Messages  bool `json:"messages"`
		Receipts  bool `json:"receipts"`
		StateRoot bool `json:"stateroot"`
	}
	type stateReport struct{
		Summary summarizedHeights `json:"summary"`
		Detail map[int64]hasState `json:"details,omitempty"`
	}

	var (
		details = make(map[int64]hasState)
		headSet bool
		head = reports[0]
		oldestMessage = &lily.StateReport{}
		oldestStateRoot = &lily.StateReport{}
	)

	for _, r := range reports {
		if verbose {
			details[r.Height] = hasState{
				Messages:  r.HasMessages,
				Receipts:  r.HasReceipts,
				StateRoot: r.HasState,
			}
		}
		if !headSet && (r.HasState && r.HasMessages && r.HasReceipts) {
			head = r
			headSet = true
		}
		if r.HasState {
			oldestStateRoot = r
		}
		if r.HasMessages {
			oldestMessage = r
		}
	}

	compiledReport := stateReport{
		Detail: details,
		Summary: summarizedHeights{
			Messages:   stateHeights{ Newest: head.Height, Oldest: oldestMessage.Height },
			StateRoots: stateHeights{ Newest: head.Height, Oldest: oldestStateRoot.Height },
		},
	}

	reportOut, err := json.Marshal(compiledReport)
	if err != nil {
		return nil, err
	}

	return reportOut, nil
}

var ChainStateInspect = &cli.Command{
	Name:  "state-inspect",
	Usage: "Returns details about each epoch's state in the local datastore",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "limit",
			Aliases: []string{"l"},
			Value: 100,
			Usage: "Limit traversal of statetree when searching for oldest state by `N` heights starting from most recent",
		},
		&cli.BoolFlag{
			Name:  "verbose",
			Aliases: []string{"v"},
			Usage: "Include detailed information about the completeness of state for all traversed height(s) starting from most recent",
		},
		&cli.BoolFlag{
			Name: "compact",
			Usage: "Print JSON without whitespace",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		report, err := lapi.FindOldestState(ctx, cctx.Int64("limit"))
		if err != nil {
			return err
		}
		sort.Slice(report, func(i, j int) bool {
			return report[i].Height > report[j].Height
		})

		out, err := marshalReport(report, cctx.Bool("verbose"))
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	},
}

var ChainStateComputeRange = &cli.Command{
	Name: "state-compute",
	Usage: "Generates the state at epoch `N`",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name: "epoch",
			Aliases: []string{"e"},
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		head, err := lapi.ChainHead(ctx)
		if err != nil {
			return err
		}
		ts, err := lapi.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(cctx.Uint64("epoch")), head.Key())
		if err != nil {
			return err
		}

		_, err = lapi.StateCompute(ctx, ts.Key())
		return err

	},
}

var ChainStateCompute = &cli.Command{
	Name: "state-compute-range",
	Usage: "Generates the state from epoch `FROM` to epoch `TO`",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name: "from",
			Required: true,
		},
		&cli.Uint64Flag{
			Name: "to",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		head, err := lapi.ChainHead(ctx)
		if err != nil {
			return err
		}
		bar := pb.StartNew(int(cctx.Uint64("to") - cctx.Uint64("from")))
		bar.ShowTimeLeft = true
		bar.ShowPercent = true
		bar.ShowSpeed = true
		bar.Units = pb.U_NO
		for i := cctx.Int64("from"); i <= cctx.Int64("to"); i++ {
			ts, err := lapi.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(i), head.Key())
			if err != nil {
				return err
			}

			_, err = lapi.StateCompute(ctx, ts.Key())
			if err != nil {
				return err
			}
			bar.Add(1)
		}
		bar.Finish()
		return nil

	},
}

var ChainHeadCmd = &cli.Command{
	Name:  "head",
	Usage: "Print chain head",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		head, err := lapi.ChainHead(ctx)
		if err != nil {
			return err
		}

		for _, c := range head.Cids() {
			fmt.Println(c)
		}
		return nil
	},
}

var ChainGetBlock = &cli.Command{
	Name:      "getblock",
	Usage:     "Get a block and print its details",
	ArgsUsage: "[blockCid]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "raw",
			Usage: "print just the raw block header",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.Args().Present() {
			return fmt.Errorf("must pass cid of block to print")
		}

		bcid, err := cid.Decode(cctx.Args().First())
		if err != nil {
			return err
		}

		blk, err := lapi.ChainGetBlock(ctx, bcid)
		if err != nil {
			return fmt.Errorf("get block failed: %w", err)
		}

		if cctx.Bool("raw") {
			out, err := json.MarshalIndent(blk, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(out))
			return nil
		}

		msgs, err := lapi.ChainGetBlockMessages(ctx, bcid)
		if err != nil {
			return fmt.Errorf("failed to get messages: %w", err)
		}

		pmsgs, err := lapi.ChainGetParentMessages(ctx, bcid)
		if err != nil {
			return fmt.Errorf("failed to get parent messages: %w", err)
		}

		recpts, err := lapi.ChainGetParentReceipts(ctx, bcid)
		if err != nil {
			log.Warn(err)
			// return fmt.Errorf("failed to get receipts: %w", err)
		}

		cblock := struct {
			types.BlockHeader
			BlsMessages    []*types.Message
			SecpkMessages  []*types.SignedMessage
			ParentReceipts []*types.MessageReceipt
			ParentMessages []cid.Cid
		}{}

		cblock.BlockHeader = *blk
		cblock.BlsMessages = msgs.BlsMessages
		cblock.SecpkMessages = msgs.SecpkMessages
		cblock.ParentReceipts = recpts
		cblock.ParentMessages = apiMsgCids(pmsgs)

		out, err := json.MarshalIndent(cblock, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	},
}

func apiMsgCids(in []api.Message) []cid.Cid {
	out := make([]cid.Cid, len(in))
	for k, v := range in {
		out[k] = v.Cid
	}
	return out
}

var ChainReadObjCmd = &cli.Command{
	Name:      "read-obj",
	Usage:     "Read the raw bytes of an object",
	ArgsUsage: "[objectCid]",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		c, err := cid.Decode(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("failed to parse cid input: %s", err)
		}

		obj, err := lapi.ChainReadObj(ctx, c)
		if err != nil {
			return err
		}

		fmt.Printf("%x\n", obj)
		return nil
	},
}

var ChainStatObjCmd = &cli.Command{
	Name:      "stat-obj",
	Usage:     "Collect size and ipld link counts for objs",
	ArgsUsage: "[cid]",
	Description: `Collect object size and ipld link count for an object.

   When a base is provided it will be walked first, and all links visisted
   will be ignored when the passed in object is walked.
`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "base",
			Usage: "ignore links found in this obj",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		obj, err := cid.Decode(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("failed to parse cid input: %s", err)
		}

		base := cid.Undef
		if cctx.IsSet("base") {
			base, err = cid.Decode(cctx.String("base"))
			if err != nil {
				return err
			}
		}

		stats, err := lapi.ChainStatObj(ctx, obj, base)
		if err != nil {
			return err
		}

		fmt.Printf("Links: %d\n", stats.Links)
		fmt.Printf("Size: %s (%d)\n", types.SizeStr(types.NewInt(stats.Size)), stats.Size)
		return nil
	},
}

var ChainGetMsgCmd = &cli.Command{
	Name:      "getmessage",
	Usage:     "Get and print a message by its cid",
	ArgsUsage: "[messageCid]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass a cid of a message to get")
		}

		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		c, err := cid.Decode(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("failed to parse cid input: %w", err)
		}

		mb, err := lapi.ChainReadObj(ctx, c)
		if err != nil {
			return fmt.Errorf("failed to read object: %w", err)
		}

		var i interface{}
		m, err := types.DecodeMessage(mb)
		if err != nil {
			sm, err := types.DecodeSignedMessage(mb)
			if err != nil {
				return fmt.Errorf("failed to decode object as a message: %w", err)
			}
			i = sm
		} else {
			i = m
		}

		enc, err := json.MarshalIndent(i, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(enc))
		return nil
	},
}

var ChainListCmd = &cli.Command{
	Name:    "list",
	Aliases: []string{"love"},
	Usage:   "View a segment of the chain",
	Flags: []cli.Flag{
		&cli.Uint64Flag{Name: "height", DefaultText: "current head"},
		&cli.IntFlag{Name: "count", Value: 30},
		&cli.StringFlag{
			Name:  "format",
			Usage: "specify the format to print out tipsets",
			Value: "<height>: (<time>) <blocks>",
		},
		&cli.BoolFlag{
			Name:  "gas-stats",
			Usage: "view gas statistics for the chain",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		var head *types.TipSet

		if cctx.IsSet("height") {
			head, err = lapi.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(cctx.Uint64("height")), types.EmptyTSK)
		} else {
			head, err = lapi.ChainHead(ctx)
		}
		if err != nil {
			return err
		}

		count := cctx.Int("count")
		if count < 1 {
			return nil
		}

		tss := make([]*types.TipSet, 0, count)
		tss = append(tss, head)

		for i := 1; i < count; i++ {
			if head.Height() == 0 {
				break
			}

			head, err = lapi.ChainGetTipSet(ctx, head.Parents())
			if err != nil {
				return err
			}

			tss = append(tss, head)
		}

		if cctx.Bool("gas-stats") {
			otss := make([]*types.TipSet, 0, len(tss))
			for i := len(tss) - 1; i >= 0; i-- {
				otss = append(otss, tss[i])
			}
			tss = otss
			for i, ts := range tss {
				pbf := ts.Blocks()[0].ParentBaseFee
				fmt.Printf("%d: %d blocks (baseFee: %s -> maxFee: %s)\n", ts.Height(), len(ts.Blocks()), ts.Blocks()[0].ParentBaseFee, types.FIL(types.BigMul(pbf, types.NewInt(uint64(lotusbuild.BlockGasLimit)))))

				for _, b := range ts.Blocks() {
					msgs, err := lapi.ChainGetBlockMessages(ctx, b.Cid())
					if err != nil {
						return err
					}
					var limitSum int64
					psum := big.NewInt(0)
					for _, m := range msgs.BlsMessages {
						limitSum += m.GasLimit
						psum = big.Add(psum, m.GasPremium)
					}

					for _, m := range msgs.SecpkMessages {
						limitSum += m.Message.GasLimit
						psum = big.Add(psum, m.Message.GasPremium)
					}

					lenmsgs := len(msgs.BlsMessages) + len(msgs.SecpkMessages)

					avgpremium := big.Zero()
					if lenmsgs > 0 {
						avgpremium = big.Div(psum, big.NewInt(int64(lenmsgs)))
					}

					fmt.Printf("\t%s: \t%d msgs, gasLimit: %d / %d (%0.2f%%), avgPremium: %s\n", b.Miner, len(msgs.BlsMessages)+len(msgs.SecpkMessages), limitSum, lotusbuild.BlockGasLimit, 100*float64(limitSum)/float64(lotusbuild.BlockGasLimit), avgpremium)
				}
				if i < len(tss)-1 {
					msgs, err := lapi.ChainGetParentMessages(ctx, tss[i+1].Blocks()[0].Cid())
					if err != nil {
						return err
					}
					var limitSum int64
					for _, m := range msgs {
						limitSum += m.Message.GasLimit
					}

					recpts, err := lapi.ChainGetParentReceipts(ctx, tss[i+1].Blocks()[0].Cid())
					if err != nil {
						return err
					}

					var gasUsed int64
					for _, r := range recpts {
						gasUsed += r.GasUsed
					}

					gasEfficiency := 100 * float64(gasUsed) / float64(limitSum)
					gasCapacity := 100 * float64(limitSum) / float64(lotusbuild.BlockGasLimit)

					fmt.Printf("\ttipset: \t%d msgs, %d (%0.2f%%) / %d (%0.2f%%)\n", len(msgs), gasUsed, gasEfficiency, limitSum, gasCapacity)
				}
				fmt.Println()
			}
		} else {
			for i := len(tss) - 1; i >= 0; i-- {
				printTipSet(cctx.String("format"), tss[i])
			}
		}
		return nil
	},
}

var ChainSetHeadCmd = &cli.Command{
	Name:      "sethead",
	Usage:     "manually set the local nodes head tipset (Caution: normally only used for recovery)",
	ArgsUsage: "[tipsetkey]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "genesis",
			Usage: "reset head to genesis",
		},
		&cli.Uint64Flag{
			Name:  "epoch",
			Usage: "reset head to given epoch",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		var ts *types.TipSet

		if cctx.Bool("genesis") {
			ts, err = lapi.ChainGetGenesis(ctx)
		}
		if ts == nil && cctx.IsSet("epoch") {
			ts, err = lapi.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(cctx.Uint64("epoch")), types.EmptyTSK)
		}
		if ts == nil {
			ts, err = parseTipSet(ctx, lapi, cctx.Args().Slice())
		}
		if err != nil {
			return err
		}

		if ts == nil {
			return fmt.Errorf("must pass cids for tipset to set as head")
		}

		if err := lapi.ChainSetHead(ctx, ts.Key()); err != nil {
			return err
		}

		return nil
	},
}

func printTipSet(format string, ts *types.TipSet) {
	format = strings.ReplaceAll(format, "<height>", fmt.Sprint(ts.Height()))
	format = strings.ReplaceAll(format, "<time>", time.Unix(int64(ts.MinTimestamp()), 0).Format(time.Stamp))
	blks := "[ "
	for _, b := range ts.Blocks() {
		blks += fmt.Sprintf("%s: %s,", b.Cid(), b.Miner)
	}
	blks += " ]"

	sCids := make([]string, 0, len(blks))

	for _, c := range ts.Cids() {
		sCids = append(sCids, c.String())
	}

	format = strings.ReplaceAll(format, "<tipset>", strings.Join(sCids, ","))
	format = strings.ReplaceAll(format, "<blocks>", blks)
	format = strings.ReplaceAll(format, "<weight>", fmt.Sprint(ts.Blocks()[0].ParentWeight))

	fmt.Println(format)
}

func parseTipSet(ctx context.Context, api lily.LilyAPI, vals []string) (*types.TipSet, error) {
	var headers []*types.BlockHeader
	for _, c := range vals {
		blkc, err := cid.Decode(c)
		if err != nil {
			return nil, err
		}

		bh, err := api.ChainGetBlock(ctx, blkc)
		if err != nil {
			return nil, err
		}

		headers = append(headers, bh)
	}

	return types.NewTipSet(headers)
}

func printSortedActorVersions(av map[actors.Version]cid.Cid) error {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Version", "Name", "Family", "Code"})
	var versions []int
	for v := range av {
		versions = append(versions, int(v))
	}
	sort.Ints(versions)
	for _, v := range versions {
		name, family, err := util.ActorNameAndFamilyFromCode(av[actors.Version(v)])
		if err != nil {
			return err
		}
		t.AppendRow(table.Row{v, name, family, av[actors.Version(v)]})
		t.AppendSeparator()
	}
	fmt.Println(t.Render())
	return nil
}

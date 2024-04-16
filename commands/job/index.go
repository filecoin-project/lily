package job

import (
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"

	"github.com/filecoin-project/lotus/chain/types"
	lotuscli "github.com/filecoin-project/lotus/cli"
)

type indexOps struct {
	height int64
	tsKey  string
	// must be set in before function
	tipsetKey types.TipSetKey
}

var indexFlags indexOps

//revive:disable
var IndexCmd = &cli.Command{
	Name:  "index",
	Usage: "Index the state of a tipset from the filecoin blockchain.",
	Description: `
The index command may be used to index a single tipset from the filecoin blockchain specified either by height or by tipset key.
`,
	Subcommands: []*cli.Command{
		IndexTipSetCmd,
		IndexHeightCmd,
	},
	Before: func(_ *cli.Context) error {
		tasks := RunFlags.Tasks.Value()
		for _, taskName := range tasks {
			if _, found := tasktype.TaskLookup[taskName]; found {
				continue
			} else if _, found := tasktype.TableLookup[taskName]; found {
				continue
			} else {
				return fmt.Errorf("unknown task: %s", taskName)
			}
		}
		return nil
	},
}

var IndexTipSetCmd = &cli.Command{
	Name:  "tipset",
	Usage: "Index the state of a tipset from the filecoin blockchain by tipset key.",
	Subcommands: []*cli.Command{
		IndexNotifyCmd,
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "tipset",
			Usage:       "TipSetKey to index",
			Destination: &indexFlags.tsKey,
			Required:    true,
		},
	},
	Before: func(_ *cli.Context) error {
		tsk, err := parseTipSetKey(indexFlags.tsKey)
		if err != nil {
			return fmt.Errorf("failed to parse tipset key: %w", err)
		}
		indexFlags.tipsetKey = tsk

		return nil
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		_, err = api.LilyIndex(ctx, &lily.LilyIndexConfig{
			JobConfig: RunFlags.ParseJobConfig("index-tipset"),
			TipSet:    indexFlags.tipsetKey,
		})
		if err != nil {
			return err
		}

		return nil
	},
}

var IndexHeightCmd = &cli.Command{
	Name:  "height",
	Usage: "Index the state of a tipset from the filecoin blockchain by height.",
	Description: `
	Index the state of a tipset from the filecoin blockchain by height. If the provided height is a null-round an error will be returned.
`,
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:        "height",
			Usage:       "Height to index",
			Destination: &indexFlags.height,
			Required:    true,
		},
	},
	Subcommands: []*cli.Command{
		IndexNotifyCmd,
	},
	Before: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		ts, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(indexFlags.height), types.EmptyTSK)
		if err != nil {
			return err
		}

		if indexFlags.height != int64(ts.Height()) {
			return fmt.Errorf("height (%d) is null round, next non-null round height: %d", indexFlags.height, ts.Height())
		}
		indexFlags.tipsetKey = ts.Key()

		return nil
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		_, err = api.LilyIndex(ctx, &lily.LilyIndexConfig{
			JobConfig: RunFlags.ParseJobConfig("index-height"),
			TipSet:    indexFlags.tipsetKey,
		})
		if err != nil {
			return err
		}

		return nil
	},
}

var IndexNotifyCmd = &cli.Command{
	Name:  "notify",
	Usage: "notify the provided queueing system of the tipset to index allowing tipset-workers to perform the indexing.",
	Description: `
The notify command will insert tasks into the provided queueing system for consumption by tipset-workers.
This command should be used when lily is configured to perform distributed indexing.
`,
	Flags: []cli.Flag{
		NotifyQueueFlag,
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		cfg := &lily.LilyIndexNotifyConfig{
			IndexConfig: lily.LilyIndexConfig{
				JobConfig: RunFlags.ParseJobConfig("index-notify"),
				TipSet:    indexFlags.tipsetKey,
			},
			Queue: notifyFlags.queue,
		}

		_, err = api.LilyIndexNotify(ctx, cfg)
		if err != nil {
			return err
		}

		return nil
	},
}

func parseTipSetKey(val string) (types.TipSetKey, error) {
	tskStr := strings.Split(val, ",")
	var cids []cid.Cid
	for _, c := range tskStr {
		blkc, err := cid.Decode(c)
		if err != nil {
			return types.EmptyTSK, err
		}
		cids = append(cids, blkc)
	}

	return types.NewTipSetKey(cids...), nil
}

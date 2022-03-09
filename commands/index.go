package commands

import (
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens/lily"
)

type indexOps struct {
	tasks    string
	storage  string
	apiAddr  string
	apiToken string
	name     string
	window   time.Duration
}

var indexFlags indexOps

var IndexTipSetCmd = &cli.Command{
	Name:  "tipset",
	Usage: "Index the state of a tipset from the filecoin blockchain by tipset key",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		var tsStr string
		if tsStr = cctx.Args().First(); tsStr == "" {
			return xerrors.Errorf("tipset argument required")
		}

		tsk, err := parseTipSetKey(tsStr)
		if err != nil {
			return xerrors.Errorf("failed to parse tipset key: %w", err)
		}

		cfg := &lily.LilyIndexConfig{
			JobConfig: jobConfigFromFlags(cctx, runFlags),
			TipSet:    tsk,
		}

		api, closer, err := GetAPI(ctx, indexFlags.apiAddr, indexFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		_, err = api.LilyIndex(ctx, cfg)
		if err != nil {
			return err
		}

		return nil
	},
}

var IndexHeightCmd = &cli.Command{
	Name:  "height",
	Usage: "Index the state of a tipset from the filecoin blockchain by height",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		var tsStr string
		if tsStr = cctx.Args().First(); tsStr == "" {
			return xerrors.Errorf("height argument required")
		}

		api, closer, err := GetAPI(ctx, indexFlags.apiAddr, indexFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		height, err := strconv.ParseInt(cctx.Args().First(), 10, 46)
		if err != nil {
			return err
		}
		ts, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(height), types.EmptyTSK)
		if err != nil {
			return err
		}

		if height != int64(ts.Height()) {
			log.Warnf("height (%d) is null round, indexing height %d", height, ts.Height())
		}

		cfg := &lily.LilyIndexConfig{
			JobConfig: jobConfigFromFlags(cctx, runFlags),
			TipSet:    ts.Key(),
		}

		_, err = api.LilyIndex(ctx, cfg)
		if err != nil {
			return err
		}

		return nil
	},
}

var IndexCmd = &cli.Command{
	Name:  "index",
	Usage: "Index the state of a tipset from the filecoin blockchain.",
	Subcommands: []*cli.Command{
		IndexTipSetCmd,
		IndexHeightCmd,
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

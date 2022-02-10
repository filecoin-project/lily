package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain"
	"github.com/filecoin-project/lily/lens/lily"
)

type indexOps struct {
	tasks    string
	storage  string
	apiAddr  string
	apiToken string
	name     string
}

var indexFlags indexOps

var IndexTipSetCmd = &cli.Command{
	Name:  "tipset",
	Usage: "Index the state of a tipset from the filecoin blockchain by tipset key",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		indexName := fmt.Sprintf("index_%d", time.Now().Unix())
		if indexFlags.name != "" {
			indexName = indexFlags.name
		}

		var tsStr string
		if tsStr = cctx.Args().First(); tsStr == "" {
			return xerrors.Errorf("tipset argument required")
		}

		tsk, err := parseTipSetKey(strings.Split(tsStr, ","))
		if err != nil {
			return xerrors.Errorf("failed to parse tipset key: %w", err)
		}

		cfg := &lily.LilyIndexConfig{
			TipSet:  tsk,
			Name:    indexName,
			Tasks:   strings.Split(indexFlags.tasks, ","),
			Storage: indexFlags.storage,
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

		indexName := fmt.Sprintf("index_%d", time.Now().Unix())
		if indexFlags.name != "" {
			indexName = indexFlags.name
		}

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
			TipSet:  ts.Key(),
			Name:    indexName,
			Tasks:   strings.Split(indexFlags.tasks, ","),
			Storage: indexFlags.storage,
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
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "tasks",
			Usage:       "Comma separated list of tasks to run. Each task is reported separately in the database.",
			EnvVars:     []string{"LILY_TASKS"},
			Value:       strings.Join([]string{chain.BlocksTask, chain.MessagesTask, chain.ChainEconomicsTask, chain.ActorStatesRawTask}, ","),
			Destination: &indexFlags.tasks,
		},
		&cli.StringFlag{
			Name:        "storage",
			Usage:       "Name of storage that results will be written to.",
			EnvVars:     []string{"LILY_STORAGE"},
			Value:       "",
			Destination: &indexFlags.storage,
		},
		&cli.StringFlag{
			Name:        "api",
			Usage:       "Address of lily api in multiaddr format.",
			EnvVars:     []string{"LILY_API"},
			Value:       "/ip4/127.0.0.1/tcp/1234",
			Destination: &indexFlags.apiAddr,
		},
		&cli.StringFlag{
			Name:        "api-token",
			Usage:       "Authentication token for lily api.",
			EnvVars:     []string{"LILY_API_TOKEN"},
			Value:       "",
			Destination: &indexFlags.apiToken,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "Name of job for easy identification later.",
			EnvVars:     []string{"LILY_JOB_NAME"},
			Value:       "",
			Destination: &indexFlags.name,
		},
	},
	Subcommands: []*cli.Command{
		IndexTipSetCmd,
		IndexHeightCmd,
	},
}

func parseTipSetKey(vals []string) (types.TipSetKey, error) {
	var cids []cid.Cid
	for _, c := range vals {
		blkc, err := cid.Decode(c)
		if err != nil {
			return types.EmptyTSK, err
		}
		cids = append(cids, blkc)
	}

	return types.NewTipSetKey(cids...), nil
}

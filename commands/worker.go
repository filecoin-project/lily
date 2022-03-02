package commands

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/lens/lily"
)

var redisFlags struct {
	network  string
	addr     string
	username string
	password string
	db       int
	poolSize int
}

var redisNetworkFlag = &cli.StringFlag{
	Name:        "redis-network",
	Usage:       "Network type to use, either tcp or unix",
	Value:       "tcp",
	Destination: &redisFlags.network,
}

var redisAddrFlag = &cli.StringFlag{
	Name:        "redis-addr",
	Usage:       `Redis server address in "host:port" format`,
	Value:       "127.0.0.1:6379",
	Destination: &redisFlags.addr,
}

var redisUsernameFlag = &cli.StringFlag{
	Name:        "redis-username",
	Usage:       `Username to authenticate the current connection when redis ACLs are used.`,
	Value:       "",
	Destination: &redisFlags.username,
}

var redisPasswordFlag = &cli.StringFlag{
	Name:        "redis-password",
	Usage:       `Password to authenticate the current connection`,
	Value:       "",
	Destination: &redisFlags.password,
}

var redisDBFlag = &cli.IntFlag{
	Name:        "redis-db",
	Usage:       `Redis DB to select after connection to server`,
	Value:       0,
	Destination: &redisFlags.db,
}

var redisPoolSizeFlag = &cli.IntFlag{
	Name:        "redis-poolsize",
	Usage:       `Maximum number of socket connection, default is 10 connections per every CPU as reported by runtime.NumCPU`,
	Value:       0,
	Destination: &redisFlags.poolSize,
}

var redisFlagSet = []cli.Flag{
	redisNetworkFlag,
	redisAddrFlag,
	redisUsernameFlag,
	redisPasswordFlag,
	redisDBFlag,
	redisPoolSizeFlag,
}

var tipSetNotifierFlags struct {
	confidence int
	name       string
}

var WorkerCmd = &cli.Command{
	Name: "worker-start",
	Subcommands: []*cli.Command{
		TipSetNotifierCmd,
		TipSetWorkerCmd,
	},
}

var TipSetNotifierCmd = &cli.Command{
	Name: "tipset-notifier",
	Flags: flagSet(
		clientAPIFlagSet,
		redisFlagSet,
		[]cli.Flag{
			&cli.IntFlag{
				Name:        "confidence",
				Usage:       "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database",
				EnvVars:     []string{"LILY_CONFIDENCE"},
				Value:       2,
				Destination: &tipSetNotifierFlags.confidence,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of job for easy identification later.",
				EnvVars:     []string{"LILY_JOB_NAME"},
				Value:       "",
				Destination: &tipSetNotifierFlags.name,
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		if tipSetNotifierFlags.name == "" {
			id, err := api.ID(ctx)
			if err != nil {
				return err
			}
			tipSetNotifierFlags.name = id.ShortString()
		}

		cfg := &lily.LilyTipSetNotifierConfig{
			Redis: &lily.LilyRedisClientConfig{
				Network:  redisFlags.network,
				Addr:     redisFlags.addr,
				Username: redisFlags.username,
				Password: redisFlags.password,
				DB:       redisFlags.db,
				PoolSize: redisFlags.poolSize,
			},
			Confidence:          tipSetNotifierFlags.confidence,
			Name:                tipSetNotifierFlags.name,
			RestartOnFailure:    true,
			RestartOnCompletion: false,
			RestartDelay:        0,
		}

		res, err := api.StartTipSetNotifier(ctx, cfg)
		if err != nil {
			return err
		}

		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}

		return nil

	},
}

var tipsetWorkerFlags struct {
	name        string
	storage     string
	concurrency int
}

var TipSetWorkerCmd = &cli.Command{
	Name: "tipset-processor",
	Flags: flagSet(
		clientAPIFlagSet,
		redisFlagSet,
		[]cli.Flag{
			&cli.IntFlag{
				Name:        "concurrency",
				Usage:       "Concurrency sets the maximum number of concurrent processing of tasks. If set to a zero or negative value it will be set to the number of CPUs usable by the current process.",
				Value:       1,
				Destination: &tipsetWorkerFlags.concurrency,
			},
			&cli.StringFlag{
				Name:        "storage",
				Usage:       "Name of storage that results will be written to.",
				EnvVars:     []string{"LILY_STORAGE"},
				Value:       "",
				Destination: &tipsetWorkerFlags.storage,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of job for easy identification later.",
				EnvVars:     []string{"LILY_JOB_NAME"},
				Value:       "",
				Destination: &tipsetWorkerFlags.name,
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		if tipsetWorkerFlags.name == "" {
			id, err := api.ID(ctx)
			if err != nil {
				return err
			}
			tipsetWorkerFlags.name = id.ShortString()
		}

		cfg := &lily.LilyTipSetWorkerConfig{
			Redis: &lily.LilyRedisClientConfig{
				Network:  redisFlags.network,
				Addr:     redisFlags.addr,
				Username: redisFlags.username,
				Password: redisFlags.password,
				DB:       redisFlags.db,
				PoolSize: redisFlags.poolSize,
			},
			Concurrency:         tipsetWorkerFlags.concurrency,
			Storage:             tipsetWorkerFlags.storage,
			Name:                tipsetWorkerFlags.name,
			RestartOnFailure:    true,
			RestartOnCompletion: false,
			RestartDelay:        0,
		}

		res, err := api.StartTipSetWorker(ctx, cfg)
		if err != nil {
			return err
		}

		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}

		return nil
	},
}

package cli

import (
	"context"
	"net/http"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	cli2 "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/chain"
	api2 "github.com/filecoin-project/sentinel-visor/node/api"
	"github.com/filecoin-project/sentinel-visor/node/api/apistruct"
)

var SentinelStartWatchCmd = &cli.Command{
	Name:  "watch",
	Usage: "control watch(s) against the chain",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name: "confidence",
		},
	},
	Action: func(cctx *cli.Context) error {
		apic, closer, err := GetSentinelNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := cli2.ReqContext(cctx)

		// TODO: add a config to Lily and inject it to set these configs
		cfg := &api2.LilyWatchConfig{
			Name:       "lily",
			Tasks:      chain.AllTheTasks,
			Window:     builtin.EpochDurationSeconds * time.Second,
			Confidence: 0,
			Database: &api2.LilyDatabaseConfig{
				URL:                  "postgres://postgres:password@localhost:5432/postgres?sslmode=disable",
				Name:                 "lily-database",
				PoolSize:             75,
				AllowUpsert:          false,
				AllowSchemaMigration: false,
			},
		}

		if err := apic.LilyWatchStart(ctx, cfg); err != nil {
			return err
		}
		// wait for user to cancel
		// <-ctx.Done()
		return nil
	},
}

func GetSentinelNodeAPI(ctx *cli.Context) (api2.LilyNode, jsonrpc.ClientCloser, error) {
	addr, headers, err := cli2.GetRawAPI(ctx, repo.FullNode)
	if err != nil {
		return nil, nil, err
	}

	return NewSentinelNodeRPC(ctx.Context, addr, headers)
}

func NewSentinelNodeRPC(ctx context.Context, addr string, requestHeader http.Header) (api2.LilyNode, jsonrpc.ClientCloser, error) {
	var res apistruct.LilyNodeStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.FullNodeStruct.Internal,
			&res.Internal,
		},
		requestHeader,
	)
	return &res, closer, err
}

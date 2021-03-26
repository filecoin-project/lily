package commands

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/lens/lily"
)

// lilyAPI is a JSON-RPC client targeting a lily node. It's initialized in a
// cli.BeforeFunc.
var (
	lilyAPI lily.LilyAPI
	Closer  jsonrpc.ClientCloser
)

func initialize(c *cli.Context) error {
	var err error
	lilyAPI, Closer, err = GetSentinelNodeAPI(c)
	if err != nil {
		return err
	}
	return nil
}

func destroy(c *cli.Context) error {
	if Closer != nil {
		Closer()
	}
	return nil
}

func GetSentinelNodeAPI(ctx *cli.Context) (lily.LilyAPI, jsonrpc.ClientCloser, error) {
	addr, headers, err := lotuscli.GetRawAPI(ctx, repo.FullNode)
	if err != nil {
		return nil, nil, err
	}

	return NewSentinelNodeRPC(ctx.Context, addr, headers)
}

func NewSentinelNodeRPC(ctx context.Context, addr string, requestHeader http.Header) (lily.LilyAPI, jsonrpc.ClientCloser, error) {
	var res lily.LilyAPIStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)
	return &res, closer, err
}

// Lily Node settings for injection into lotus node.
func LilyNodeAPIOption(out *lily.LilyAPI, fopts ...node.Option) node.Option {
	resAPI := &lily.LilyNodeAPI{}
	opts := node.Options(
		node.WithRepoType(repo.FullNode),
		node.Options(fopts...),
		node.WithInvokesKey(node.ExtractApiKey, resAPI),
	)
	*out = resAPI
	return opts
}

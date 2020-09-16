package lotus

import (
	"context"
	"fmt"
	"github.com/filecoin-project/sentinel-visor/lens"
	"net/http"
	"strings"

	"github.com/urfave/cli/v2"

	logging "github.com/ipfs/go-log/v2"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/filecoin-project/go-jsonrpc"
	lotus_api "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	lcli "github.com/filecoin-project/lotus/cli"
)

var log = logging.Logger("visor/lens/lotus")

type APICloser jsonrpc.ClientCloser

func GetFullNodeAPI(cctx *cli.Context) (context.Context, lens.API, APICloser, error) {
	var api lotus_api.FullNode
	var closer jsonrpc.ClientCloser
	var err error

	if tokenMaddr := cctx.String("api"); tokenMaddr != "" {
		toks := strings.Split(tokenMaddr, ":")
		if len(toks) != 2 {
			return nil, nil, nil, fmt.Errorf("invalid api tokens, expected <token>:<maddr>, got: %s", tokenMaddr)
		}

		api, closer, err = getFullNodeAPIUsingCredentials(cctx.Context, toks[1], toks[0])
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		api, closer, err = lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	ctx := lcli.ReqContext(cctx)

	v, err := api.Version(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	log.Infof("Lotus API version: %s", v.Version)

	lensAPI := NewAPIWrapper(api, NewCacheCtxStore(ctx, api, 10_000))

	return ctx, lensAPI, APICloser(closer), nil
}

func getFullNodeAPIUsingCredentials(ctx context.Context, listenAddr, token string) (lotus_api.FullNode, jsonrpc.ClientCloser, error) {
	parsedAddr, err := ma.NewMultiaddr(listenAddr)
	if err != nil {
		return nil, nil, err
	}

	_, addr, err := manet.DialArgs(parsedAddr)
	if err != nil {
		return nil, nil, err
	}

	return client.NewFullNodeRPC(ctx, apiURI(addr), apiHeaders(token))
}

func apiURI(addr string) string {
	return "ws://" + addr + "/rpc/v0"
}
func apiHeaders(token string) http.Header {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)
	return headers
}

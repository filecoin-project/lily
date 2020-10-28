package lotus

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/filecoin-project/go-jsonrpc"
	lotus_api "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	lcli "github.com/filecoin-project/lotus/cli"
	lru "github.com/hashicorp/golang-lru"
	logging "github.com/ipfs/go-log/v2"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
)

var log = logging.Logger("visor/lens/lotus")

type APIOpener struct {
	// TODO: replace dependency on cli.Context with tokenMaddr and repo path
	cctx  *cli.Context
	cache *lru.ARCCache // cache shared across all instances of the api
}

func NewAPIOpener(cctx *cli.Context, cacheSize int) (*APIOpener, lens.APICloser, error) {
	ac, err := lru.NewARC(cacheSize)
	if err != nil {
		return nil, nil, xerrors.Errorf("new arc cache: %w", err)
	}

	o := &APIOpener{
		cctx:  cctx,
		cache: ac,
	}

	return o, lens.APICloser(func() {}), nil
}

func (o *APIOpener) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	var api lotus_api.FullNode
	var closer jsonrpc.ClientCloser
	var err error

	if tokenMaddr := o.cctx.String("api"); tokenMaddr != "" {
		toks := strings.Split(tokenMaddr, ":")
		if len(toks) != 2 {
			return nil, nil, fmt.Errorf("invalid api tokens, expected <token>:<maddr>, got: %s", tokenMaddr)
		}

		api, closer, err = getFullNodeAPIUsingCredentials(ctx, toks[1], toks[0])
		if err != nil {
			return nil, nil, xerrors.Errorf("get full node api with credentials: %w", err)
		}
	} else {
		api, closer, err = lcli.GetFullNodeAPI(o.cctx)
		if err != nil {
			return nil, nil, xerrors.Errorf("get full node api: %w", err)
		}
	}

	cacheStore, err := NewCacheCtxStore(ctx, api, o.cache)
	if err != nil {
		return nil, nil, xerrors.Errorf("new cache store: %w", err)
	}

	lensAPI := NewAPIWrapper(api, cacheStore)

	return lensAPI, lens.APICloser(closer), nil
}

func getFullNodeAPIUsingCredentials(ctx context.Context, listenAddr, token string) (lotus_api.FullNode, jsonrpc.ClientCloser, error) {
	parsedAddr, err := ma.NewMultiaddr(listenAddr)
	if err != nil {
		return nil, nil, xerrors.Errorf("parse listen address: %w", err)
	}

	_, addr, err := manet.DialArgs(parsedAddr)
	if err != nil {
		return nil, nil, xerrors.Errorf("dial multiaddress: %w", err)
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

package commands

import (
	"context"
	"net/http"
	"strings"

	"github.com/filecoin-project/go-jsonrpc"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/repo"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens/lily"
)

func GetAPI(ctx context.Context, addrStr string, token string) (lily.LilyAPI, jsonrpc.ClientCloser, error) {
	addrStr = strings.TrimSpace(addrStr)

	ainfo := cliutil.APIInfo{Addr: addrStr, Token: []byte(token)}

	addr, err := ainfo.DialArgs()
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	return NewSentinelNodeRPC(ctx, addr, ainfo.AuthHeader())
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

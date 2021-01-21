package camera

import (
	"bytes"
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/sentinel-visor/lens"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipld/go-car"
	"github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
	"io"
	"net/http"
	"strings"
	"sync"
)

func NewFilm() *Film {
	return &Film{
		captured: make(map[cid.Cid]struct{}),
	}
}

type Film struct {
	captured   map[cid.Cid]struct{}
	capturedMu sync.Mutex
}

func (f *Film) Capture(cids ...cid.Cid) {
	f.capturedMu.Lock()
	defer f.capturedMu.Unlock()
	for _, c := range cids {
		f.captured[c] = struct{}{}
	}
}

func NewCameraOpener(cctx *cli.Context, cacheSize int) (*CameraOpener, lens.APICloser, error) {
	ac, err := lru.NewARC(cacheSize)
	if err != nil {
		return nil, nil, xerrors.Errorf("new arc cache: %w", err)
	}

	var rawaddr, rawtoken string

	if cctx.IsSet("api") {
		tokenMaddr := cctx.String("api")
		toks := strings.Split(tokenMaddr, ":")
		if len(toks) != 2 {
			return nil, nil, fmt.Errorf("invalid api tokens, expected <token>:<maddr>, got: %s", tokenMaddr)
		}

		rawtoken = toks[0]
		rawaddr = toks[1]
	} else if cctx.IsSet("repo") {
		repoPath := cctx.String("repo")
		p, err := homedir.Expand(repoPath)
		if err != nil {
			return nil, nil, xerrors.Errorf("expand home dir (%s): %w", repoPath, err)
		}

		r, err := repo.NewFS(p)
		if err != nil {
			return nil, nil, xerrors.Errorf("open repo at path: %s; %w", p, err)
		}

		ma, err := r.APIEndpoint()
		if err != nil {
			return nil, nil, xerrors.Errorf("api endpoint: %w", err)
		}

		token, err := r.APIToken()
		if err != nil {
			return nil, nil, xerrors.Errorf("api token: %w", err)
		}

		rawaddr = ma.String()
		rawtoken = string(token)
	} else {
		return nil, nil, xerrors.Errorf("cannot connect to lotus api: missing --api or --repo flags")
	}

	parsedAddr, err := ma.NewMultiaddr(rawaddr)
	if err != nil {
		return nil, nil, xerrors.Errorf("parse listen address: %w", err)
	}

	_, addr, err := manet.DialArgs(parsedAddr)
	if err != nil {
		return nil, nil, xerrors.Errorf("dial multiaddress: %w", err)
	}

	o := &CameraOpener{
		cache:   ac,
		film:    NewFilm(),
		addr:    apiURI(addr),
		headers: apiHeaders(rawtoken),
	}

	return o, lens.APICloser(func() {}), nil
}

type CameraOpener struct {
	film *Film

	cache   *lru.ARCCache // cache shared across all instances of the api
	addr    string
	headers http.Header
}

func (c *CameraOpener) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	api, closer, err := client.NewFullNodeRPC(ctx, c.addr, c.headers)
	if err != nil {
		return nil, nil, xerrors.Errorf("new full node rpc: %w", err)
	}

	cacheStore, err := NewCacheCtxStore(ctx, api, c.cache)
	if err != nil {
		return nil, nil, xerrors.Errorf("new cache store: %w", err)
	}

	lensAPI := NewAPIRecorder(api, cacheStore, c.film)

	return lensAPI, lens.APICloser(closer), nil
}

func (c *CameraOpener) Record(ctx context.Context, w io.Writer, roots ...cid.Cid) error {
	carWalkFn := func(nd format.Node) (out []*format.Link, err error) {
		for _, link := range nd.Links() {
			if _, ok := c.film.captured[link.Cid]; !ok {
				continue
			}
			if link.Cid.Prefix().Codec == cid.FilCommitmentSealed || link.Cid.Prefix().Codec == cid.FilCommitmentUnsealed {
				continue
			}
			out = append(out, link)
		}
		return out, nil
	}

	api, close, err := c.Open(ctx)
	if err != nil {
		return err
	}
	defer close()

	bs := &apiBlockstore{api: api}
	var (
		offl    = offline.Exchange(bs)
		blkserv = blockservice.New(bs, offl)
		dserv   = merkledag.NewDAGService(blkserv)
	)

	return car.WriteCarWithWalker(ctx, dserv, roots, w, carWalkFn)
}

func apiURI(addr string) string {
	return "ws://" + addr + "/rpc/v0"
}

func apiHeaders(token string) http.Header {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)
	return headers
}

type CacheCtxStore struct {
	cache *lru.ARCCache
	ctx   context.Context
	api   api.FullNode
}

func NewCacheCtxStore(ctx context.Context, api api.FullNode, cache *lru.ARCCache) (*CacheCtxStore, error) {
	return &CacheCtxStore{
		cache: cache,
		ctx:   ctx,
		api:   api,
	}, nil
}

func (cs *CacheCtxStore) Context() context.Context {
	return cs.ctx
}

func (cs *CacheCtxStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	ctx, span := global.Tracer("").Start(ctx, "CacheCtxStore.Get")
	defer span.End()
	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return fmt.Errorf("out parameter does not implement CBORUnmarshaler")
	}

	// hit :)
	v, hit := cs.cache.Get(c)
	if hit {
		return cu.UnmarshalCBOR(bytes.NewReader(v.([]byte)))
	}

	// miss :(
	raw, err := cs.api.ChainReadObj(ctx, c)
	if err != nil {
		return xerrors.Errorf("read obj: %w", err)
	}

	if err := cu.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
		return xerrors.Errorf("unmarshal obj: %w", err)
	}

	cs.cache.Add(c, raw)
	return nil
}

func (cs *CacheCtxStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return cid.Undef, fmt.Errorf("put is not implemented on CacheCtxStore")
}

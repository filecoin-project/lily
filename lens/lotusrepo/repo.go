package lotusrepo

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/lib/cachebs"
	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
)

type RepoAPI struct {
	api.FullNode
	context.Context
	cacheSize int
}

func (ra *RepoAPI) ComputeGasOutputs(gasUsed, gasLimit int64, baseFee, feeCap, gasPremium abi.TokenAmount) vm.GasOutputs {
	return vm.ComputeGasOutputs(gasUsed, gasLimit, baseFee, feeCap, gasPremium)
}

func (ra *RepoAPI) Store() adt.Store {
	i, ok := ra.FullNode.(*impl.FullNodeAPI)
	if !ok {
		return nil
	}
	store := i.ChainAPI.Chain.Blockstore()
	cachedStore := cachebs.NewBufferedBstore(store, ra.cacheSize)
	cs := cbor.NewCborStore(cachedStore)
	adtStore := adt.WrapStore(ra.Context, cs)
	return adtStore
}

func GetAPI(c *cli.Context) (context.Context, lens.API, lens.APICloser, error) {
	rapi := RepoAPI{}

	r, err := repo.NewFS(c.String("repo"))
	if err != nil {
		return nil, nil, nil, err
	}

	options := []node.Option{
		node.FullAPI(&rapi.FullNode),
		node.Repo(r),
	}
	stop, err := node.New(c.Context, options...)
	if err != nil {
		return nil, nil, nil, err
	}

	sf := func() {
		fmt.Printf("%v", stop(c.Context))
	}

	rapi.Context = c.Context
	rapi.cacheSize = c.Int("lens-cache-hint")
	return c.Context, &rapi, sf, nil
}

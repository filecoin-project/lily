package vector

import (
	"context"
	"fmt"
	"io"

	"github.com/urfave/cli/v2"

	"github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	cbor "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	car "github.com/ipld/go-car"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/lib/bufbstore"
	"github.com/filecoin-project/lotus/lib/ulimit"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
)

func NewAPIOpener(c *cli.Context) (*APIOpener, lens.APICloser, error) {
	capi := CaptureAPI{}

	if _, _, err := ulimit.ManageFdLimit(); err != nil {
		return nil, nil, fmt.Errorf("setting file descriptor limit: %s", err)
	}

	r, err := repo.NewFS(c.String("repo"))
	if err != nil {
		return nil, nil, err
	}

	exists, err := r.Exists()
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, fmt.Errorf("lotus repo doesn't exist")
	}

	var lr repo.LockedRepo
	if c.Bool("repo-read-only") {
		lr, err = r.LockRO(repo.FullNode)
	} else {
		lr, err = r.Lock(repo.FullNode)
	}
	if err != nil {
		return nil, nil, err
	}

	sf := func() {
		lr.Close()
	}

	bs, err := lr.Blockstore(c.Context, repo.BlockstoreChain)
	if err != nil {
		return nil, nil, err
	}

	// wrap the repos blockstore with a tracing implementation capturing all cid's read.
	capi.tbs = NewTracingBlockstore(bs)

	mds, err := lr.Datastore(c.Context, "/metadata")
	if err != nil {
		return nil, nil, err
	}

	cs := store.NewChainStore(capi.tbs, capi.tbs, mds, vm.Syscalls(&fakeVerifier{}), journal.NilJournal())
	if err := cs.Load(); err != nil {
		return nil, nil, err
	}

	sm := stmgr.NewStateManager(cs)

	capi.FullNodeAPI.ChainAPI.Chain = cs
	capi.FullNodeAPI.ChainAPI.ChainModuleAPI = &full.ChainModule{Chain: cs}
	capi.FullNodeAPI.StateAPI.Chain = cs
	capi.FullNodeAPI.StateAPI.StateManager = sm
	capi.FullNodeAPI.StateAPI.StateModuleAPI = &full.StateModule{Chain: cs, StateManager: sm}

	capi.Context = c.Context
	capi.cacheSize = c.Int("lens-cache-hint")
	return &APIOpener{capi: &capi}, sf, nil
}

type APIOpener struct {
	capi *CaptureAPI
}

func (o *APIOpener) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	return o.capi, lens.APICloser(func() {}), nil
}
func (c *APIOpener) CaptureAsCAR(ctx context.Context, w io.Writer, roots ...cid.Cid) error {
	carWalkFn := func(nd format.Node) (out []*format.Link, err error) {
		for _, link := range nd.Links() {
			if _, ok := c.capi.tbs.traced[link.Cid]; !ok {
				continue
			}
			if link.Cid.Prefix().Codec == cid.FilCommitmentSealed || link.Cid.Prefix().Codec == cid.FilCommitmentUnsealed {
				continue
			}
			out = append(out, link)
		}
		return out, nil
	}

	var (
		offl    = offline.Exchange(c.capi.tbs)
		blkserv = blockservice.New(c.capi.tbs, offl)
		dserv   = merkledag.NewDAGService(blkserv)
	)

	return car.WriteCarWithWalker(ctx, dserv, roots, w, carWalkFn)
}

type CaptureAPI struct {
	impl.FullNodeAPI
	context.Context
	cacheSize int

	tbs *TracingBlockstore
}

func (c *CaptureAPI) Store() adt.Store {
	cachedStore := bufbstore.NewBufferedBstore(c.tbs)
	cs := cbor.NewCborStore(cachedStore)
	adtStore := adt.WrapStore(c.Context, cs)
	return adtStore
}

func (c *CaptureAPI) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	return util.GetExecutedMessagesForTipset(ctx, c.FullNodeAPI.ChainAPI.Chain, ts, pts)
}

func (c *CaptureAPI) StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	act, err := lens.OptimizedStateGetActorWithFallback(ctx, c.ChainAPI.Chain.Store(ctx), c.ChainAPI, c.StateAPI, addr, tsk)
	if err != nil {
		return nil, err
	}
	//c.tbs.Record(act.Head)
	return act, nil
}

// From https://github.com/ribasushi/ltsh/blob/5b0211033020570217b0ae37b50ee304566ac218/cmd/lotus-shed/deallifecycles.go#L41-L171
type fakeVerifier struct{}

var _ ffiwrapper.Verifier = (*fakeVerifier)(nil)

func (m fakeVerifier) VerifySeal(svi proof.SealVerifyInfo) (bool, error) {
	return true, nil
}

func (m fakeVerifier) VerifyWinningPoSt(ctx context.Context, info proof.WinningPoStVerifyInfo) (bool, error) {
	return true, nil
}

func (m fakeVerifier) VerifyWindowPoSt(ctx context.Context, info proof.WindowPoStVerifyInfo) (bool, error) {
	return true, nil
}

func (m fakeVerifier) GenerateWinningPoStSectorChallenge(ctx context.Context, proof abi.RegisteredPoStProof, id abi.ActorID, randomness abi.PoStRandomness, u uint64) ([]uint64, error) {
	panic("GenerateWinningPoStSectorChallenge not supported")
}

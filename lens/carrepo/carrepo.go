package carrepo

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/sentinel-visor/lens"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/lib/bufbstore"
	"github.com/filecoin-project/lotus/lib/ulimit"
	marketevents "github.com/filecoin-project/lotus/markets/loggers"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/willscott/carbs"
)

type APIOpener struct {
	// shared instance of the repo since the opener holds an exclusive lock on it
	rapi *CarAPI
}

func NewAPIOpener(c *cli.Context) (*APIOpener, lens.APICloser, error) {
	rapi := CarAPI{}

	if _, _, err := ulimit.ManageFdLimit(); err != nil {
		return nil, nil, fmt.Errorf("setting file descriptor limit: %s", err)
	}

	db, err := carbs.Load(c.String("repo"), false)
	if err != nil {
		return nil, nil, err
	}
	cacheDB := NewCachingStore(db)

	r := repo.NewMemory(nil)

	lr, err := r.Lock(repo.FullNode)
	if err != nil {
		return nil, nil, err
	}

	mds, err := lr.Datastore("/metadata")
	if err != nil {
		return nil, nil, err
	}

	cs := store.NewChainStore(cacheDB, cacheDB, mds, vm.Syscalls(&fakeVerifier{}), journal.NilJournal())

	headKey, err := db.Roots()
	if err != nil {
		return nil, nil, err
	}

	headTs, err := cs.LoadTipSet(types.NewTipSetKey(headKey...))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load our own chainhead: %w", err)
	}
	if err := cs.SetHead(headTs); err != nil {
		return nil, nil, fmt.Errorf("failed to set our own chainhead: %w", err)
	}

	sm := stmgr.NewStateManager(cs)

	rapi.FullNodeAPI.ChainAPI.Chain = cs
	rapi.FullNodeAPI.ChainAPI.ChainModuleAPI = &full.ChainModule{Chain: cs}
	rapi.FullNodeAPI.StateAPI.Chain = cs
	rapi.FullNodeAPI.StateAPI.StateManager = sm
	rapi.FullNodeAPI.StateAPI.StateModuleAPI = &full.StateModule{Chain: cs, StateManager: sm}

	sf := func() {
		lr.Close()
	}

	rapi.Context = c.Context
	rapi.cacheSize = c.Int("lens-cache-hint")
	return &APIOpener{&rapi}, sf, nil
}

func (o *APIOpener) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	return o.rapi, lens.APICloser(func() {}), nil
}

type CarAPI struct {
	impl.FullNodeAPI
	context.Context
	cacheSize int
}

func (ra *CarAPI) ComputeGasOutputs(gasUsed, gasLimit int64, baseFee, feeCap, gasPremium abi.TokenAmount) vm.GasOutputs {
	return vm.ComputeGasOutputs(gasUsed, gasLimit, baseFee, feeCap, gasPremium)
}

func (ra *CarAPI) Store() adt.Store {
	bs := ra.FullNodeAPI.ChainAPI.Chain.Blockstore()
	bufferedStore := bufbstore.NewBufferedBstore(bs)
	cs := cbor.NewCborStore(bufferedStore)
	adtStore := adt.WrapStore(ra.Context, cs)
	return adtStore
}

// TODO: Remove. See https://github.com/filecoin-project/sentinel-visor/issues/196
func (ra *CarAPI) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	return lens.OptimizedStateGetActorWithFallback(ctx, ra.ChainAPI.Chain, ra.FullNodeAPI, actor, tsk)
}

func (ra *CarAPI) ClientStartDeal(ctx context.Context, params *api.StartDealParams) (*cid.Cid, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientListDeals(ctx context.Context) ([]api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientGetDealInfo(ctx context.Context, d cid.Cid) (*api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientGetDealUpdates(ctx context.Context) (<-chan api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientHasLocal(ctx context.Context, root cid.Cid) (bool, error) {
	return false, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientFindData(ctx context.Context, root cid.Cid, piece *cid.Cid) ([]api.QueryOffer, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientMinerQueryOffer(ctx context.Context, miner address.Address, root cid.Cid, piece *cid.Cid) (api.QueryOffer, error) {
	return api.QueryOffer{}, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientImport(ctx context.Context, ref api.FileRef) (*api.ImportRes, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientRemoveImport(ctx context.Context, importID multistore.StoreID) error {
	return fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientImportLocal(ctx context.Context, f io.Reader) (cid.Cid, error) {
	return cid.Undef, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientListImports(ctx context.Context) ([]api.Import, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientRetrieve(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) error {
	return fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientRetrieveWithEvents(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) (<-chan marketevents.RetrievalEvent, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*storagemarket.StorageAsk, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientCalcCommP(ctx context.Context, inpath string) (*api.CommPRet, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientDealSize(ctx context.Context, root cid.Cid) (api.DataSize, error) {
	return api.DataSize{}, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientGenCar(ctx context.Context, ref api.FileRef, outputPath string) error {
	return fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientListDataTransfers(ctx context.Context) ([]api.DataTransferChannel, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientDataTransferUpdates(ctx context.Context) (<-chan api.DataTransferChannel, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *CarAPI) ClientRetrieveTryRestartInsufficientFunds(ctx context.Context, paymentChannel address.Address) error {
	return fmt.Errorf("unsupported")
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

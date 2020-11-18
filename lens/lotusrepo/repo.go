package lotusrepo

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
)

type APIOpener struct {
	// shared instance of the repo since the opener holds an exclusive lock on it
	rapi *RepoAPI
}

func NewAPIOpener(c *cli.Context) (*APIOpener, lens.APICloser, error) {
	rapi := RepoAPI{}

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

	lr, err := r.LockRO(repo.FullNode)
	if err != nil {
		return nil, nil, err
	}

	bs, err := lr.Blockstore(repo.BlockstoreChain)
	if err != nil {
		return nil, nil, err
	}

	mds, err := lr.Datastore("/metadata")
	if err != nil {
		return nil, nil, err
	}

	cs := store.NewChainStore(bs, bs, mds, vm.Syscalls(&fakeVerifier{}), journal.NilJournal())
	if err := cs.Load(); err != nil {
		return nil, nil, err
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
	return &APIOpener{rapi: &rapi}, sf, nil
}

func (o *APIOpener) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	return o.rapi, lens.APICloser(func() {}), nil
}

type RepoAPI struct {
	impl.FullNodeAPI
	context.Context
	cacheSize int
}

func (ra *RepoAPI) ComputeGasOutputs(gasUsed, gasLimit int64, baseFee, feeCap, gasPremium abi.TokenAmount) vm.GasOutputs {
	return vm.ComputeGasOutputs(gasUsed, gasLimit, baseFee, feeCap, gasPremium)
}

func (ra *RepoAPI) Store() adt.Store {
	bs := ra.FullNodeAPI.ChainAPI.Chain.Blockstore()
	cachedStore := bufbstore.NewBufferedBstore(bs)
	cs := cbor.NewCborStore(cachedStore)
	adtStore := adt.WrapStore(ra.Context, cs)
	return adtStore
}

func (ra *RepoAPI) ClientStartDeal(ctx context.Context, params *api.StartDealParams) (*cid.Cid, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientListDeals(ctx context.Context) ([]api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientGetDealInfo(ctx context.Context, d cid.Cid) (*api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientGetDealUpdates(ctx context.Context) (<-chan api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientHasLocal(ctx context.Context, root cid.Cid) (bool, error) {
	return false, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientFindData(ctx context.Context, root cid.Cid, piece *cid.Cid) ([]api.QueryOffer, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientMinerQueryOffer(ctx context.Context, miner address.Address, root cid.Cid, piece *cid.Cid) (api.QueryOffer, error) {
	return api.QueryOffer{}, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientImport(ctx context.Context, ref api.FileRef) (*api.ImportRes, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientRemoveImport(ctx context.Context, importID multistore.StoreID) error {
	return fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientImportLocal(ctx context.Context, f io.Reader) (cid.Cid, error) {
	return cid.Undef, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientListImports(ctx context.Context) ([]api.Import, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientRetrieve(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) error {
	return fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientRetrieveWithEvents(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) (<-chan marketevents.RetrievalEvent, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*storagemarket.StorageAsk, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientCalcCommP(ctx context.Context, inpath string) (*api.CommPRet, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientDealSize(ctx context.Context, root cid.Cid) (api.DataSize, error) {
	return api.DataSize{}, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientGenCar(ctx context.Context, ref api.FileRef, outputPath string) error {
	return fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientListDataTransfers(ctx context.Context) ([]api.DataTransferChannel, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientDataTransferUpdates(ctx context.Context) (<-chan api.DataTransferChannel, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *RepoAPI) ClientRetrieveTryRestartInsufficientFunds(ctx context.Context, paymentChannel address.Address) error {
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

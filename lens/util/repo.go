package util

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/lib/blockstore"
	"github.com/filecoin-project/lotus/lib/bufbstore"
	"github.com/filecoin-project/lotus/lib/ulimit"
	marketevents "github.com/filecoin-project/lotus/markets/loggers"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

type APIOpener struct {
	// shared instance of the repo since the opener holds an exclusive lock on it
	rapi *LensAPI
}

type HeadMthd func(ctx context.Context, lookback int) (*types.TipSetKey, error)

func NewAPIOpener(c *cli.Context, bs blockstore.Blockstore, head HeadMthd) (*APIOpener, lens.APICloser, error) {
	rapi := LensAPI{}

	if _, _, err := ulimit.ManageFdLimit(); err != nil {
		return nil, nil, fmt.Errorf("setting file descriptor limit: %s", err)
	}
	r := repo.NewMemory(nil)

	lr, err := r.Lock(repo.FullNode)
	if err != nil {
		return nil, nil, err
	}

	mds, err := lr.Datastore("/metadata")
	if err != nil {
		return nil, nil, err
	}

	cs := store.NewChainStore(bs, bs, mds, vm.Syscalls(&fakeVerifier{}), journal.NilJournal())

	const safetyLookBack = 5

	headKey, err := head(c.Context, safetyLookBack)
	if err != nil {
		return nil, nil, err
	}

	headTs, err := cs.LoadTipSet(*headKey)
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
	return &APIOpener{rapi: &rapi}, sf, nil
}

func (o *APIOpener) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	return o.rapi, lens.APICloser(func() {}), nil
}

type LensAPI struct {
	impl.FullNodeAPI
	context.Context
	cacheSize int
	cs        *store.ChainStore
}

// TODO: Remove. See https://github.com/filecoin-project/sentinel-visor/issues/196
func (ra *LensAPI) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	return lens.OptimizedStateGetActorWithFallback(ctx, ra.cs.Store(ctx), ra.FullNodeAPI.ChainAPI, ra.FullNodeAPI.StateAPI, actor, tsk)
}

func (ra *LensAPI) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	return GetExecutedMessagesForTipset(ctx, ra.cs, ts, pts)
}

func (ra *LensAPI) Store() adt.Store {
	bs := ra.FullNodeAPI.ChainAPI.Chain.Blockstore()
	bufferedStore := bufbstore.NewBufferedBstore(bs)
	cs := cbor.NewCborStore(bufferedStore)
	adtStore := adt.WrapStore(ra.Context, cs)
	return adtStore
}

func (ra *LensAPI) ClientStartDeal(ctx context.Context, params *api.StartDealParams) (*cid.Cid, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientListDeals(ctx context.Context) ([]api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientGetDealInfo(ctx context.Context, d cid.Cid) (*api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientGetDealUpdates(ctx context.Context) (<-chan api.DealInfo, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientHasLocal(ctx context.Context, root cid.Cid) (bool, error) {
	return false, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientFindData(ctx context.Context, root cid.Cid, piece *cid.Cid) ([]api.QueryOffer, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientMinerQueryOffer(ctx context.Context, miner address.Address, root cid.Cid, piece *cid.Cid) (api.QueryOffer, error) {
	return api.QueryOffer{}, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientImport(ctx context.Context, ref api.FileRef) (*api.ImportRes, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientRemoveImport(ctx context.Context, importID multistore.StoreID) error {
	return fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientImportLocal(ctx context.Context, f io.Reader) (cid.Cid, error) {
	return cid.Undef, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientListImports(ctx context.Context) ([]api.Import, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientRetrieve(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) error {
	return fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientRetrieveWithEvents(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) (<-chan marketevents.RetrievalEvent, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*storagemarket.StorageAsk, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientCalcCommP(ctx context.Context, inpath string) (*api.CommPRet, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientDealSize(ctx context.Context, root cid.Cid) (api.DataSize, error) {
	return api.DataSize{}, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientGenCar(ctx context.Context, ref api.FileRef, outputPath string) error {
	return fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientListDataTransfers(ctx context.Context) ([]api.DataTransferChannel, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientDataTransferUpdates(ctx context.Context) (<-chan api.DataTransferChannel, error) {
	return nil, fmt.Errorf("unsupported")
}

func (ra *LensAPI) ClientRetrieveTryRestartInsufficientFunds(ctx context.Context, paymentChannel address.Address) error {
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

// GetMessagesForTipset returns a list of messages sent as part of pts (parent) with receipts found in ts (child).
// No attempt at deduplication of messages is made.
func GetExecutedMessagesForTipset(ctx context.Context, cs *store.ChainStore, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	if !types.CidArrsEqual(ts.Parents().Cids(), pts.Cids()) {
		return nil, xerrors.Errorf("child tipset (%s) is not on the same chain as parent (%s)", ts.Key(), pts.Key())
	}

	stateTree, err := state.LoadStateTree(cs.Store(ctx), ts.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load state tree: %w", err)
	}

	parentStateTree, err := state.LoadStateTree(cs.Store(ctx), pts.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load state tree: %w", err)
	}

	// Build a lookup of actor codes
	actorCodes := map[address.Address]cid.Cid{}
	if err := stateTree.ForEach(func(a address.Address, act *types.Actor) error {
		actorCodes[a] = act.Code
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("iterate actors: %w", err)
	}

	getActorCode := func(a address.Address) cid.Cid {
		c, ok := actorCodes[a]
		if ok {
			return c
		}

		return cid.Undef
	}

	// Build a lookup of which blocks each message appears in
	messageBlocks := map[cid.Cid][]cid.Cid{}
	for blockIdx, bh := range pts.Blocks() {
		blscids, secpkcids, err := cs.ReadMsgMetaCids(bh.Messages)
		if err != nil {
			return nil, xerrors.Errorf("read messages for block: %w", err)
		}

		for _, c := range blscids {
			messageBlocks[c] = append(messageBlocks[c], pts.Cids()[blockIdx])
		}

		for _, c := range secpkcids {
			messageBlocks[c] = append(messageBlocks[c], pts.Cids()[blockIdx])
		}

	}

	bmsgs, err := cs.BlockMsgsForTipset(pts)
	if err != nil {
		return nil, xerrors.Errorf("block messages for tipset: %w", err)
	}

	pblocks := pts.Blocks()
	if len(bmsgs) != len(pblocks) {
		// logic error somewhere
		return nil, xerrors.Errorf("mismatching number of blocks returned from block messages, got %d wanted %d", len(bmsgs), len(pblocks))
	}

	count := 0
	for _, bm := range bmsgs {
		count += len(bm.BlsMessages) + len(bm.SecpkMessages)
	}

	// Start building a list of completed message with receipt
	emsgs := make([]*lens.ExecutedMessage, 0, count)

	// bmsgs is ordered by block
	var index uint64
	for blockIdx, bm := range bmsgs {
		for _, blsm := range bm.BlsMessages {
			msg := blsm.VMMessage()
			emsgs = append(emsgs, &lens.ExecutedMessage{
				Cid:           blsm.Cid(),
				Height:        pts.Height(),
				Message:       msg,
				BlockHeader:   pblocks[blockIdx],
				Blocks:        messageBlocks[blsm.Cid()],
				Index:         index,
				FromActorCode: getActorCode(msg.From),
				ToActorCode:   getActorCode(msg.To),
			})
			index++
		}

		for _, secm := range bm.SecpkMessages {
			msg := secm.VMMessage()
			emsgs = append(emsgs, &lens.ExecutedMessage{
				Cid:           secm.Cid(),
				Height:        pts.Height(),
				Message:       secm.VMMessage(),
				BlockHeader:   pblocks[blockIdx],
				Blocks:        messageBlocks[secm.Cid()],
				Index:         index,
				FromActorCode: getActorCode(msg.From),
				ToActorCode:   getActorCode(msg.To),
			})
			index++
		}

	}

	// Retrieve receipts using a block from the child tipset
	rs, err := adt.AsArray(cs.Store(ctx), ts.Blocks()[0].ParentMessageReceipts)
	if err != nil {
		return nil, xerrors.Errorf("amt load: %w", err)
	}

	if rs.Length() != uint64(len(emsgs)) {
		// logic error somewhere
		return nil, xerrors.Errorf("mismatching number of receipts: got %d wanted %d", rs.Length(), len(emsgs))
	}

	// Create a skeleton vm just for calling ShouldBurn
	vmi, err := vm.NewVM(ctx, &vm.VMOpts{
		StateBase: pts.ParentState(),
		Epoch:     pts.Height(),
		Bstore:    cs.Blockstore(),
	})
	if err != nil {
		return nil, xerrors.Errorf("creating temporary vm: %w", err)
	}

	// Receipts are in same order as BlockMsgsForTipset
	for _, em := range emsgs {
		var r types.MessageReceipt
		if found, err := rs.Get(em.Index, &r); err != nil {
			return nil, err
		} else if !found {
			return nil, xerrors.Errorf("failed to find receipt %d", em.Index)
		}
		em.Receipt = &r

		burn, err := vmi.ShouldBurn(parentStateTree, em.Message, em.Receipt.ExitCode)
		if err != nil {
			return nil, xerrors.Errorf("deciding whether should burn failed: %w", err)
		}

		em.GasOutputs = vm.ComputeGasOutputs(em.Receipt.GasUsed, em.Message.GasLimit, em.BlockHeader.ParentBaseFee, em.Message.GasFeeCap, em.Message.GasPremium, burn)

	}

	return emsgs, nil
}

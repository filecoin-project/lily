package actorstate_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	bstore "github.com/filecoin-project/lotus/lib/blockstore"
	sa0account "github.com/filecoin-project/specs-actors/actors/builtin/account"
	sa0init "github.com/filecoin-project/specs-actors/actors/builtin/init"
	samarket "github.com/filecoin-project/specs-actors/actors/builtin/market"
	sa0miner "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	sa0power "github.com/filecoin-project/specs-actors/actors/builtin/power"
	sa0reward "github.com/filecoin-project/specs-actors/actors/builtin/reward"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	sa2init "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	sa2power "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	sa2reward "github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/ipld/go-car"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/testutil"
)

var VectorsDir string

func init() {
	VectorsDir = "/home/frrist/src/github.com/filecoin-project/sentinel-visor/tasks/actorstate/vectors"
}

var _ actorstate.ActorStateAPI = (*MockAPI)(nil)

type MockAPI struct {
	t       testing.TB
	actors  map[actorKey]*types.Actor
	miners  map[actorKey]miner.State
	tipsets map[types.TipSetKey]*types.TipSet
	bs      bstore.Blockstore
	store   adt.Store
}

func NewMockAPI(test testing.TB) *MockAPI {
	bs := bstore.NewTemporarySync()
	return &MockAPI{
		t:       test,
		bs:      bs,
		tipsets: make(map[types.TipSetKey]*types.TipSet),
		miners:  make(map[actorKey]miner.State),
		actors:  make(map[actorKey]*types.Actor),
		store:   adt.WrapStore(context.Background(), cbornode.NewCborStore(bs)),
	}
}

type actorKey struct {
	tsk  types.TipSetKey
	addr address.Address
}

func (m *MockAPI) Store() adt.Store {
	return m.store
}

func (m *MockAPI) ChainHasObj(ctx context.Context, c cid.Cid) (bool, error) {
	return m.bs.Has(c)
}

func (m *MockAPI) ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error) {
	blk, err := m.bs.Get(c)
	if err != nil {
		return nil, xerrors.Errorf("blockstore get: %w", err)
	}

	return blk.RawData(), nil
}

func (m *MockAPI) ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error) {
	return &api.BlockMessages{
		BlsMessages:   []*types.Message{},
		SecpkMessages: []*types.SignedMessage{},
		Cids:          []cid.Cid{},
	}, nil
}

func (m *MockAPI) ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	return m.tipsets[tsk], nil
}

func (m *MockAPI) StateReadState(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	act, err := m.StateGetActor(ctx, actor, tsk)
	if err != nil {
		return nil, xerrors.Errorf("getting actor: %w", err)
	}

	var state interface{}
	if err := m.store.Get(ctx, act.Head, &state); err != nil {
		return nil, xerrors.Errorf("getting actor head: %w", err)
	}

	return &api.ActorState{
		Balance: act.Balance,
		State:   state,
	}, nil
}

func (m *MockAPI) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	key := actorKey{
		tsk:  tsk,
		addr: actor,
	}
	act, ok := m.actors[key]
	if !ok {
		return nil, xerrors.Errorf("actor not found addr:%s tsk=%s", actor, tsk)
	}
	return act, nil
}

func (m *MockAPI) StateGetReceipt(ctx context.Context, bcid cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error) {
	panic("not implemented")
}

func (m *MockAPI) StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	panic("not implemented yet")
}

func (m *MockAPI) StateMinerSectors(ctx context.Context, a address.Address, field *bitfield.BitField, key types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	state, ok := m.miners[actorKey{
		tsk:  key,
		addr: a,
	}]
	if !ok {
		return nil, xerrors.New("Miner not found")
	}
	return state.LoadSectors(field)
}

// ----------------- MockAPI Helpers ----------------------------

type FakeTsOpts func(bh *types.BlockHeader)

func WithHeight(h int64) FakeTsOpts {
	return func(bh *types.BlockHeader) {
		bh.Height = abi.ChainEpoch(h)
	}
}

func (m *MockAPI) fakeTipset(minerAddr address.Address, timestamp uint64, opts ...FakeTsOpts) *types.TipSet {
	bh := &types.BlockHeader{
		Miner:                 minerAddr,
		Height:                5,
		ParentStateRoot:       testutil.RandomCid(),
		Messages:              testutil.RandomCid(),
		ParentMessageReceipts: testutil.RandomCid(),
		BlockSig:              &crypto.Signature{Type: crypto.SigTypeBLS},
		BLSAggregate:          &crypto.Signature{Type: crypto.SigTypeBLS},
		Timestamp:             timestamp,
	}
	for _, opt := range opts {
		opt(bh)
	}
	ts, err := types.NewTipSet([]*types.BlockHeader{bh})
	require.NoError(m.t, err)

	m.tipsets[ts.Key()] = ts
	return ts
}

func (m *MockAPI) setActor(tsk types.TipSetKey, addr address.Address, actor *types.Actor) {
	key := actorKey{
		tsk:  tsk,
		addr: addr,
	}
	m.actors[key] = actor
}

func (m *MockAPI) mustCreateMarketState(ctx context.Context, deals map[abi.DealID]*samarket.DealState, props map[abi.DealID]*samarket.DealProposal, balances map[address.Address]balance) cid.Cid {
	dealRootCid := m.mustCreateDealAMT(deals)

	propRootCid := m.mustCreateProposalAMT(props)

	balancesCids := m.mustCreateBalanceTable(balances)

	state := m.mustCreateEmptyMarketState()

	state.States = dealRootCid
	state.Proposals = propRootCid
	state.EscrowTable = balancesCids[0]
	state.LockedTable = balancesCids[1]

	stateCid, err := m.store.Put(ctx, state)
	require.NoError(m.t, err)

	return stateCid
}

func (m *MockAPI) mustCreateEmptyMarketState() *samarket.State {
	emptyArrayCid, err := adt.MakeEmptyArray(m.store).Root()
	require.NoError(m.t, err)

	emptyMap, err := adt.MakeEmptyMap(m.store).Root()
	require.NoError(m.t, err)

	return samarket.ConstructState(emptyArrayCid, emptyMap, emptyMap)
}

func (m *MockAPI) mustCreateDealAMT(deals map[abi.DealID]*samarket.DealState) cid.Cid {
	root := adt.MakeEmptyArray(m.store)
	for dealID, dealState := range deals {
		err := root.Set(uint64(dealID), dealState)
		require.NoError(m.t, err)
	}
	rootCid, err := root.Root()
	require.NoError(m.t, err)

	return rootCid
}

func (m *MockAPI) mustCreateProposalAMT(props map[abi.DealID]*samarket.DealProposal) cid.Cid {
	root := adt.MakeEmptyArray(m.store)
	for dealID, prop := range props {
		err := root.Set(uint64(dealID), prop)
		require.NoError(m.t, err)
	}
	rootCid, err := root.Root()
	require.NoError(m.t, err)

	return rootCid
}

func (m *MockAPI) mustCreateBalanceTable(balances map[address.Address]balance) [2]cid.Cid {
	escrowMapRoot := adt.MakeEmptyMap(m.store)
	escrowMapRootCid, err := escrowMapRoot.Root()
	require.NoError(m.t, err)

	escrowRoot, err := adt.AsBalanceTable(m.store, escrowMapRootCid)
	require.NoError(m.t, err)

	lockedMapRoot := adt.MakeEmptyMap(m.store)
	lockedMapRootCid, err := lockedMapRoot.Root()
	require.NoError(m.t, err)

	lockedRoot, err := adt.AsBalanceTable(m.store, lockedMapRootCid)
	require.NoError(m.t, err)

	for addr, balance := range balances {
		err := escrowRoot.Add(addr, big.Add(balance.available, balance.locked))
		require.NoError(m.t, err)

		err = lockedRoot.Add(addr, balance.locked)
		require.NoError(m.t, err)

	}
	escrowRootCid, err := escrowRoot.Root()
	require.NoError(m.t, err)

	lockedRootCid, err := lockedRoot.Root()
	require.NoError(m.t, err)

	return [2]cid.Cid{escrowRootCid, lockedRootCid}
}

func (m *MockAPI) mustCreateEmptyPowerStateV0() *sa0power.State {
	emptyClaimsMap, err := adt.MakeEmptyMap(m.store).Root()
	require.NoError(m.t, err)

	cronEventQueueMMap, err := adt.MakeEmptyMultimap(m.store).Root()
	require.NoError(m.t, err)

	return sa0power.ConstructState(emptyClaimsMap, cronEventQueueMMap)
}

func (m *MockAPI) mustCreateEmptyPowerStateV2() *sa2power.State {
	emptyClaimsMap, err := adt.MakeEmptyMap(m.store).Root()
	require.NoError(m.t, err)

	cronEventQueueMMap, err := adt.MakeEmptyMultimap(m.store).Root()
	require.NoError(m.t, err)

	return sa2power.ConstructState(emptyClaimsMap, cronEventQueueMMap)
}

func (m *MockAPI) mustCreateEmptyRewardStateV0(currRealizedPower abi.StoragePower) *sa0reward.State {
	return sa0reward.ConstructState(currRealizedPower)
}

func (m *MockAPI) mustCreateEmptyRewardStateV2(currRealizedPower abi.StoragePower) *sa2reward.State {
	return sa2reward.ConstructState(currRealizedPower)
}

func (m *MockAPI) mustCreateEmptyInitStateV0() *sa0init.State {
	emptyMap, err := adt.MakeEmptyMap(m.store).Root()
	require.NoError(m.t, err)

	return sa0init.ConstructState(emptyMap, "visor-testing")
}

func (m *MockAPI) mustCreateEmptyInitStateV2() *sa2init.State {
	emptyMap, err := adt2.MakeEmptyMap(m.store).Root()
	require.NoError(m.t, err)

	return sa2init.ConstructState(emptyMap, "visor-testing")
}

func (m *MockAPI) mustCreateEmptyMinerStateV0() *sa0miner.State {
	ctx := context.TODO()
	emptyMap, err := adt.MakeEmptyMap(m.store).Root()
	require.NoError(m.t, err)

	emptyArray, err := adt.MakeEmptyArray(m.store).Root()
	require.NoError(m.t, err)

	emptyBitfield := bitfield.NewFromSet(nil)
	emptyBitfieldCid, err := m.store.Put(ctx, emptyBitfield)
	require.NoError(m.t, err)

	emptyDeadline := sa0miner.ConstructDeadline(emptyArray)
	emptyDeadlineCid, err := m.store.Put(ctx, emptyDeadline)
	require.NoError(m.t, err)

	emptyDeadlines := sa0miner.ConstructDeadlines(emptyDeadlineCid)
	emptyVestingFunds := sa0miner.ConstructVestingFunds()
	emptyDeadlinesCid, err := m.store.Put(ctx, emptyDeadlines)
	require.NoError(m.t, err)

	emptyVestingFundsCid, err := m.store.Put(ctx, emptyVestingFunds)
	require.NoError(m.t, err)

	ownerAddr := tutils.NewIDAddr(m.t, 123)
	workerAddr := ownerAddr
	controlAddrs := []address.Address{ownerAddr, workerAddr}
	info, err := sa0miner.ConstructMinerInfo(ownerAddr, workerAddr, controlAddrs, nil, nil, abi.RegisteredSealProof_StackedDrg64GiBV1)
	require.NoError(m.t, err)

	infoCid, err := m.store.Put(ctx, info)
	require.NoError(m.t, err)

	state, err := sa0miner.ConstructState(infoCid, 0, emptyBitfieldCid, emptyArray, emptyMap, emptyDeadlinesCid, emptyVestingFundsCid)
	require.NoError(m.t, err)

	return state
}

func (m *MockAPI) mustCreateAccountStateV0(addr address.Address) *sa0account.State {
	return &sa0account.State{Address: addr}
}

type ImportedActor struct {
	Address string `json:"address"`
	TipSet  string `json:"tipset"`
	Code    string `json:"code"`
	Head    string `json:"head"`
	Balance string `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

func (a *ImportedActor) AsActorType() *types.Actor {
	code, err := cid.Decode(a.Code)
	if err != nil {
		panic(err)
	}

	head, err := cid.Decode(a.Head)
	if err != nil {
		panic(err)
	}

	balance, err := types.BigFromString(a.Balance)
	if err != nil {
		panic(err)
	}

	return &types.Actor{
		Code:    code,
		Head:    head,
		Nonce:   a.Nonce,
		Balance: balance,
	}
}

type ActorVector struct {
	Address address.Address
	TipSet  types.TipSetKey
	Actor   *types.Actor
}

type MinerVector struct {
	Info  ActorVector
	State miner.State
}

func (m *MockAPI) MinerVectorForHead(headStr string, tipset *types.TipSet) *MinerVector {
	headCID, err := cid.Decode(headStr)
	require.NoError(m.t, err)

	actorF, err := actorFileForHead(headCID)
	require.NoError(m.t, err)
	defer actorF.Close()

	actorB, err := ioutil.ReadAll(actorF)
	require.NoError(m.t, err)

	importedActor := new(ImportedActor)
	err = json.Unmarshal(actorB, importedActor)
	require.NoError(m.t, err)

	addr, err := address.NewFromString(importedActor.Address)
	require.NoError(m.t, err)

	stateF, err := stateFileForHead(headCID)
	require.NoError(m.t, err)
	defer stateF.Close()

	header, err := car.LoadCar(m.bs, stateF)
	require.NoError(m.t, err)
	require.Len(m.t, header.Roots, 1)
	require.Equal(m.t, headCID, header.Roots[0])

	state, err := miner.Load(m.store, importedActor.AsActorType())
	require.NoError(m.t, err)

	ak := actorKey{
		tsk:  tipset.Key(),
		addr: addr,
	}
	m.actors[ak] = importedActor.AsActorType()
	m.miners[ak] = state
	return &MinerVector{
		Info: ActorVector{
			Address: addr,
			TipSet:  tipset.Key(),
			Actor:   importedActor.AsActorType(),
		},
		State: state,
	}
}

func actorFileForHead(head cid.Cid) (*os.File, error) {
	return os.Open(fmt.Sprintf("%s/%s/actor.json", VectorsDir, head))
}

func stateFileForHead(head cid.Cid) (*os.File, error) {
	return os.Open(fmt.Sprintf("%s/%s/state.car", VectorsDir, head))
}

func (m *MockAPI) MinerV0StateFromVector(tsk types.TipSetKey, addr address.Address, info types.Actor) miner.State {
	f, err := os.Open(fmt.Sprintf("%s/%s.car", VectorsDir, info.Head))
	require.NoError(m.t, err)

	header, err := car.LoadCar(m.bs, f)
	require.NoError(m.t, err)
	require.Len(m.t, header.Roots, 1)
	require.Equal(m.t, info.Head, header.Roots[0])

	state, err := miner.Load(m.store, &info)
	require.NoError(m.t, err)

	m.miners[actorKey{
		tsk:  tsk,
		addr: addr,
	}] = state

	return state
}

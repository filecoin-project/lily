package actorstate_test

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	bstore "github.com/filecoin-project/lotus/blockstore"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	sa0account "github.com/filecoin-project/specs-actors/actors/builtin/account"
	sa0init "github.com/filecoin-project/specs-actors/actors/builtin/init"
	samarket "github.com/filecoin-project/specs-actors/actors/builtin/market"
	sa0power "github.com/filecoin-project/specs-actors/actors/builtin/power"
	sa0reward "github.com/filecoin-project/specs-actors/actors/builtin/reward"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	sa2init "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	sa2power "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	sa2reward "github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	sa3init "github.com/filecoin-project/specs-actors/v3/actors/builtin/init"
	sa3power "github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	sa3reward "github.com/filecoin-project/specs-actors/v3/actors/builtin/reward"
	sa4init "github.com/filecoin-project/specs-actors/v4/actors/builtin/init"
	sa4power "github.com/filecoin-project/specs-actors/v4/actors/builtin/power"
	sa4reward "github.com/filecoin-project/specs-actors/v4/actors/builtin/reward"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/testutil"
)

var _ actorstate.ActorStateAPI = (*MockAPI)(nil)

type MockAPI struct {
	t       testing.TB
	actors  map[actorKey]*types.Actor
	tipsets map[types.TipSetKey]*types.TipSet
	bs      bstore.Blockstore
	store   adt.Store
}

func NewMockAPI(test testing.TB) *MockAPI {
	bs := bstore.NewMemorySync()
	return &MockAPI{
		t:       test,
		bs:      bs,
		tipsets: make(map[types.TipSetKey]*types.TipSet),
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

func (m *MockAPI) ChainGetParentMessages(ctx context.Context, msg cid.Cid) ([]api.Message, error) {
	return []api.Message{}, nil
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
	panic("not implemented yet")
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

func (m *MockAPI) mustCreateEmptyPowerStateV3() *sa3power.State {
	st, err := sa3power.ConstructState(m.store)
	require.NoError(m.t, err)
	return st
}

func (m *MockAPI) mustCreateEmptyPowerStateV4() *sa4power.State {
	st, err := sa4power.ConstructState(m.store)
	require.NoError(m.t, err)
	return st
}

func (m *MockAPI) mustCreateEmptyRewardStateV0(currRealizedPower abi.StoragePower) *sa0reward.State {
	return sa0reward.ConstructState(currRealizedPower)
}

func (m *MockAPI) mustCreateEmptyRewardStateV2(currRealizedPower abi.StoragePower) *sa2reward.State {
	return sa2reward.ConstructState(currRealizedPower)
}

func (m *MockAPI) mustCreateEmptyRewardStateV3(currRealizedPower abi.StoragePower) *sa3reward.State {
	return sa3reward.ConstructState(currRealizedPower)
}

func (m *MockAPI) mustCreateEmptyRewardStateV4(currRealizedPower abi.StoragePower) *sa4reward.State {
	return sa4reward.ConstructState(currRealizedPower)
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

func (m *MockAPI) mustCreateEmptyInitStateV3() *sa3init.State {
	st, err := sa3init.ConstructState(m.store, "visor-testing")
	require.NoError(m.t, err)
	return st
}

func (m *MockAPI) mustCreateEmptyInitStateV4() *sa4init.State {
	st, err := sa4init.ConstructState(m.store, "visor-testing")
	require.NoError(m.t, err)
	return st
}

func (m *MockAPI) mustCreateAccountStateV0(addr address.Address) *sa0account.State {
	return &sa0account.State{Address: addr}
}

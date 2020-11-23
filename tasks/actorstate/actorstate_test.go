package actorstate_test

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	bstore "github.com/filecoin-project/lotus/lib/blockstore"
	samarket "github.com/filecoin-project/specs-actors/actors/builtin/market"
	sa0power "github.com/filecoin-project/specs-actors/actors/builtin/power"
	sa0reward "github.com/filecoin-project/specs-actors/actors/builtin/reward"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	sa2power "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	sa2reward "github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/testutil"
)

type MockTsOpts func(bh *types.BlockHeader)

func WithHeight(h int64) MockTsOpts {
	return func(bh *types.BlockHeader) {
		bh.Height = abi.ChainEpoch(h)
	}
}

func mockTipset(minerAddr address.Address, timestamp uint64, opts ...MockTsOpts) (*types.TipSet, error) {
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
	return types.NewTipSet([]*types.BlockHeader{bh})
}

var _ actorstate.ActorStateAPI = (*MockAPI)(nil)

type MockAPI struct {
	actors  map[actorKey]*types.Actor
	tipsets map[types.TipSetKey]*types.TipSet
	bs      bstore.Blockstore
	store   adt.Store
}

func NewMockAPI() *MockAPI {
	bs := bstore.NewTemporarySync()
	return &MockAPI{
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

func (m *MockAPI) ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error) {
	return &api.BlockMessages{
		BlsMessages:   []*types.Message{},
		SecpkMessages: []*types.SignedMessage{},
		Cids:          []cid.Cid{},
	}, nil
}

func (m *MockAPI) ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	return m.tipsets[tsk], nil
	blks := make([]*types.BlockHeader, len(tsk.Cids()))
	for i, cid := range tsk.Cids() {
		if err := m.store.Get(ctx, cid, blks[i]); err != nil {
			return nil, err
		}
	}
	return types.NewTipSet(blks)
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

func (m *MockAPI) setActor(tsk types.TipSetKey, addr address.Address, actor *types.Actor) {
	key := actorKey{
		tsk:  tsk,
		addr: addr,
	}
	m.actors[key] = actor
}

func (m *MockAPI) putTipSet(ts *types.TipSet) {
	m.tipsets[ts.Key()] = ts
}

func (m *MockAPI) createMarketState(ctx context.Context, deals map[abi.DealID]*samarket.DealState, props map[abi.DealID]*samarket.DealProposal, balances map[address.Address]balance) (cid.Cid, error) {
	dealRootCid, err := m.createDealAMT(deals)
	if err != nil {
		return cid.Undef, err
	}

	propRootCid, err := m.createProposalAMT(props)
	if err != nil {
		return cid.Undef, err
	}

	balancesCids, err := m.createBalanceTable(balances)
	if err != nil {
		return cid.Undef, err
	}
	state, err := m.newEmptyMarketState()
	if err != nil {
		return cid.Undef, err
	}

	state.States = dealRootCid
	state.Proposals = propRootCid
	state.EscrowTable = balancesCids[0]
	state.LockedTable = balancesCids[1]

	stateCid, err := m.store.Put(ctx, state)
	if err != nil {
		return cid.Undef, err
	}

	return stateCid, nil
}

func (m *MockAPI) newEmptyMarketState() (*samarket.State, error) {
	emptyArrayCid, err := adt.MakeEmptyArray(m.store).Root()
	if err != nil {
		return nil, err
	}
	emptyMap, err := adt.MakeEmptyMap(m.store).Root()
	if err != nil {
		return nil, err
	}
	return samarket.ConstructState(emptyArrayCid, emptyMap, emptyMap), nil
}

func (m *MockAPI) createDealAMT(deals map[abi.DealID]*samarket.DealState) (cid.Cid, error) {
	root := adt.MakeEmptyArray(m.store)
	for dealID, dealState := range deals {
		err := root.Set(uint64(dealID), dealState)
		if err != nil {
			return cid.Undef, err
		}
	}
	rootCid, err := root.Root()
	if err != nil {
		return cid.Undef, err
	}
	return rootCid, nil
}

func (m *MockAPI) createProposalAMT(props map[abi.DealID]*samarket.DealProposal) (cid.Cid, error) {
	root := adt.MakeEmptyArray(m.store)
	for dealID, prop := range props {
		err := root.Set(uint64(dealID), prop)
		if err != nil {
			return cid.Undef, err
		}
	}
	rootCid, err := root.Root()
	if err != nil {
		return cid.Undef, err
	}
	return rootCid, nil
}

func (m *MockAPI) createBalanceTable(balances map[address.Address]balance) ([2]cid.Cid, error) {
	escrowMapRoot := adt.MakeEmptyMap(m.store)
	escrowMapRootCid, err := escrowMapRoot.Root()
	if err != nil {
		return [2]cid.Cid{}, err
	}
	escrowRoot, err := adt.AsBalanceTable(m.store, escrowMapRootCid)
	if err != nil {
		return [2]cid.Cid{}, err
	}

	lockedMapRoot := adt.MakeEmptyMap(m.store)
	lockedMapRootCid, err := lockedMapRoot.Root()
	if err != nil {
		return [2]cid.Cid{}, err
	}

	lockedRoot, err := adt.AsBalanceTable(m.store, lockedMapRootCid)
	if err != nil {
		return [2]cid.Cid{}, err
	}

	for addr, balance := range balances {
		err := escrowRoot.Add(addr, big.Add(balance.available, balance.locked))
		if err != nil {
			return [2]cid.Cid{}, err
		}

		err = lockedRoot.Add(addr, balance.locked)
		if err != nil {
			return [2]cid.Cid{}, err
		}

	}
	escrowRootCid, err := escrowRoot.Root()
	if err != nil {
		return [2]cid.Cid{}, err
	}

	lockedRootCid, err := lockedRoot.Root()
	if err != nil {
		return [2]cid.Cid{}, err
	}

	return [2]cid.Cid{escrowRootCid, lockedRootCid}, nil
}

func (m *MockAPI) newEmptyPowerStateV0() (*sa0power.State, error) {
	emptyClaimsMap, err := adt.MakeEmptyMap(m.store).Root()
	if err != nil {
		return nil, err
	}
	cronEventQueueMMap, err := adt.MakeEmptyMultimap(m.store).Root()
	if err != nil {
		return nil, err
	}
	return sa0power.ConstructState(emptyClaimsMap, cronEventQueueMMap), nil
}

func (m *MockAPI) newEmptyPowerStateV2() (*sa2power.State, error) {
	emptyClaimsMap, err := adt.MakeEmptyMap(m.store).Root()
	if err != nil {
		return nil, err
	}
	cronEventQueueMMap, err := adt.MakeEmptyMultimap(m.store).Root()
	if err != nil {
		return nil, err
	}
	return sa2power.ConstructState(emptyClaimsMap, cronEventQueueMMap), nil
}

func (m *MockAPI) newEmptyRewardStateV0(currRealizedPower abi.StoragePower) (*sa0reward.State, error) {
	return sa0reward.ConstructState(currRealizedPower), nil
}

func (m *MockAPI) newEmptyRewardStateV2(currRealizedPower abi.StoragePower) (*sa2reward.State, error) {
	return sa2reward.ConstructState(currRealizedPower), nil
}

package miner_test

import (
	"sort"
	"testing"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	minertypes "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	minerstatemocks "github.com/filecoin-project/lily/chain/actors/builtin/miner/mocks"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	minerex "github.com/filecoin-project/lily/tasks/actorstate/miner"
	"github.com/filecoin-project/lily/tasks/actorstate/miner/extraction/mocks"
	"github.com/filecoin-project/lily/testutil"
)

func TestExtractMinerPreCommitEvents(t *testing.T) {
	minerContext := new(mocks.MockMinerState)
	ts := testutil.MustFakeTipSet(t, 10)
	addr := testutil.MustMakeAddress(t, 100)
	minerContext.On("CurrentTipSet").Return(ts)
	minerContext.On("Address").Return(addr)
	fakePrecommitChanges := generateFakeSectorPreCommitChanges(10, 5)
	result := minerex.ExtractMinerPreCommitEvents(minerContext, fakePrecommitChanges)
	require.NotNil(t, result)
	require.Len(t, result, 10)
	for i, res := range result {
		require.Equal(t, addr.String(), res.MinerID)
		require.Equal(t, int64(ts.Height()), res.Height)
		require.Equal(t, ts.ParentState().String(), res.StateRoot)
		require.Equal(t, minermodel.PreCommitAdded, res.Event)
		require.EqualValues(t, fakePrecommitChanges.Added[i].Info.SectorNumber, res.SectorID)
	}
}

func TestExtractMinerSectorStateEvents(t *testing.T) {
	minerContext := new(mocks.MockMinerState)
	parentMinerState := new(minerstatemocks.State)
	currentMinerState := new(minerstatemocks.State)
	ts := testutil.MustFakeTipSet(t, 10)
	addr := testutil.MustMakeAddress(t, 100)
	minerContext.On("ParentState").Return(parentMinerState)
	minerContext.On("CurrentState").Return(currentMinerState)
	minerContext.On("CurrentTipSet").Return(ts)
	minerContext.On("Address").Return(addr)
	numTerminated := uint64(2)
	numRecovered := uint64(3)
	numFaulted := uint64(4)
	numRecovering := uint64(5)
	// 2 removed sectors, 3 recovered, 4 faulted, and 5 recovering, total of 14 sector events
	fakeSectorStateChanges := generateFakeSectorStateChanges(numTerminated, numRecovered, numFaulted, numRecovering)
	currentMinerState.On("LoadSectors", mock.Anything).Return([]*miner.SectorOnChainInfo{}, nil)
	result, err := minerex.ExtractMinerSectorStateEvents(minerContext, fakeSectorStateChanges)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, int(numTerminated+numRecovered+numFaulted+numRecovering))

	events := make(map[string][]*minermodel.MinerSectorEvent)
	for _, res := range result {
		events[res.Event] = append(events[res.Event], res)
	}

	terminated := events[minermodel.SectorTerminated]
	sort.Slice(terminated, func(i, j int) bool {
		return terminated[i].SectorID < terminated[j].SectorID
	})
	for i, e := range terminated {
		require.Equal(t, minermodel.SectorTerminated, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	recovered := events[minermodel.SectorRecovered]
	sort.Slice(recovered, func(i, j int) bool {
		return recovered[i].SectorID < recovered[j].SectorID
	})
	for i, e := range recovered {
		require.Equal(t, minermodel.SectorRecovered, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	faulted := events[minermodel.SectorFaulted]
	sort.Slice(faulted, func(i, j int) bool {
		return faulted[i].SectorID < faulted[j].SectorID
	})
	for i, e := range faulted {
		require.Equal(t, minermodel.SectorFaulted, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	recovering := events[minermodel.SectorRecovering]
	sort.Slice(recovering, func(i, j int) bool {
		return recovering[i].SectorID < recovering[j].SectorID
	})
	for i, e := range recovering {
		require.Equal(t, minermodel.SectorRecovering, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}
}

func TestExtractMinerSectorEvents(t *testing.T) {
	minerContext := new(mocks.MockMinerState)
	ts := testutil.MustFakeTipSet(t, 10)
	addr := testutil.MustMakeAddress(t, 100)
	minerContext.On("CurrentTipSet").Return(ts)
	minerContext.On("Address").Return(addr)
	sectorChanges := &miner.SectorChanges{
		// 4 sectors added, 0 and 1 contains deals
		Added: generateFakeSectorOnChainInfos(map[uint64][]abi.DealID{
			0: {0, 1, 2},
			1: {4, 5, 6},
			2: {},
			3: {},
		}),
		// 2 sectors extended
		Extended: []miner.SectorModification{
			{
				From: generateFakeSectorOnChainInfo(0),
				To:   generateFakeSectorOnChainInfo(0),
			},
			{
				From: generateFakeSectorOnChainInfo(1),
				To:   generateFakeSectorOnChainInfo(1),
			},
		},
		// 1 sector snapped
		Snapped: []miner.SectorModification{
			{
				From: generateFakeSectorOnChainInfo(0),
				To:   generateFakeSectorOnChainInfo(0),
			},
		},
		// 6 sectors removed, this value is ignored as sectors can be removed for a variety of reasons.
		// sector removal is tested in TestExtractMinerSectorStateEvents
		Removed: generateFakeSectorOnChainInfos(map[uint64][]abi.DealID{
			0: {},
			1: {},
			2: {},
			3: {},
			4: {},
			5: {},
		}),
	}
	result := minerex.ExtractMinerSectorEvents(minerContext, sectorChanges)
	require.NotNil(t, result)
	require.Len(t, result, 7)

	events := make(map[string][]*minermodel.MinerSectorEvent)
	for _, res := range result {
		events[res.Event] = append(events[res.Event], res)
	}

	added := events[minermodel.SectorAdded]
	sort.Slice(added, func(i, j int) bool {
		return added[i].SectorID < added[j].SectorID
	})
	for i, e := range added {
		require.Equal(t, minermodel.SectorAdded, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	cc := events[minermodel.CommitCapacityAdded]
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].SectorID < cc[j].SectorID
	})
	for i, e := range cc {
		require.Equal(t, minermodel.CommitCapacityAdded, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i+2, e.SectorID)
	}

	extended := events[minermodel.SectorExtended]
	sort.Slice(extended, func(i, j int) bool {
		return extended[i].SectorID < extended[j].SectorID
	})
	for i, e := range extended {
		require.Equal(t, minermodel.SectorExtended, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	snapped := events[minermodel.SectorSnapped]
	sort.Slice(snapped, func(i, j int) bool {
		return snapped[i].SectorID < snapped[j].SectorID
	})
	for i, e := range snapped {
		require.Equal(t, minermodel.SectorSnapped, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}
}

func TestExtractSectorEvents(t *testing.T) {
	minerContext := new(mocks.MockMinerState)
	parentMinerState := new(minerstatemocks.State)
	currentMinerState := new(minerstatemocks.State)
	ts := testutil.MustFakeTipSet(t, 10)
	addr := testutil.MustMakeAddress(t, 100)
	minerContext.On("ParentState").Return(parentMinerState)
	minerContext.On("CurrentState").Return(currentMinerState)
	minerContext.On("CurrentTipSet").Return(ts)
	minerContext.On("Address").Return(addr)
	numPrecommitChanges := uint64(10)
	fakePrecommitChanges := generateFakeSectorPreCommitChanges(numPrecommitChanges, 5)
	numSectorChanges := uint64(7)
	sectorChanges := &miner.SectorChanges{
		// 4 sectors added, 0 and 1 contains deals
		Added: generateFakeSectorOnChainInfos(map[uint64][]abi.DealID{
			0: {0, 1, 2},
			1: {4, 5, 6},
			2: {},
			3: {},
		}),
		// 2 sectors extended
		Extended: []miner.SectorModification{
			{
				From: generateFakeSectorOnChainInfo(0),
				To:   generateFakeSectorOnChainInfo(0),
			},
			{
				From: generateFakeSectorOnChainInfo(1),
				To:   generateFakeSectorOnChainInfo(1),
			},
		},
		// 1 sector snapped
		Snapped: []miner.SectorModification{
			{
				From: generateFakeSectorOnChainInfo(0),
				To:   generateFakeSectorOnChainInfo(0),
			},
		},
		// 6 sectors removed, this value is ignored as sectors can be removed for a variety of reasons.
		// sector removal is tested in TestExtractMinerSectorStateEvents
		Removed: generateFakeSectorOnChainInfos(map[uint64][]abi.DealID{
			0: {},
			1: {},
			2: {},
			3: {},
			4: {},
			5: {},
		}),
	}
	numTerminated := uint64(2)
	numRecovered := uint64(3)
	numFaulted := uint64(4)
	numRecovering := uint64(5)
	// 2 removed sectors, 3 recovered, 4 faulted, and 5 recovering, total of 15 sector events
	fakeSectorStateChanges := generateFakeSectorStateChanges(numTerminated, numRecovered, numFaulted, numRecovering)
	currentMinerState.On("LoadSectors", mock.Anything).Return([]*miner.SectorOnChainInfo{}, nil)

	result, err := minerex.ExtractSectorEvents(minerContext, sectorChanges, fakePrecommitChanges, fakeSectorStateChanges)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, int(numPrecommitChanges+numSectorChanges+numTerminated+numRecovered+numFaulted+numRecovering))

	events := make(map[string][]*minermodel.MinerSectorEvent)
	for _, res := range result {
		events[res.Event] = append(events[res.Event], res)
	}

	added := events[minermodel.SectorAdded]
	sort.Slice(added, func(i, j int) bool {
		return added[i].SectorID < added[j].SectorID
	})
	for i, e := range added {
		require.Equal(t, minermodel.SectorAdded, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	cc := events[minermodel.CommitCapacityAdded]
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].SectorID < cc[j].SectorID
	})
	for i, e := range cc {
		require.Equal(t, minermodel.CommitCapacityAdded, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i+2, e.SectorID)
	}

	extended := events[minermodel.SectorExtended]
	sort.Slice(extended, func(i, j int) bool {
		return extended[i].SectorID < extended[j].SectorID
	})
	for i, e := range extended {
		require.Equal(t, minermodel.SectorExtended, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	snapped := events[minermodel.SectorSnapped]
	sort.Slice(snapped, func(i, j int) bool {
		return snapped[i].SectorID < snapped[j].SectorID
	})
	for i, e := range snapped {
		require.Equal(t, minermodel.SectorSnapped, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	terminated := events[minermodel.SectorTerminated]
	sort.Slice(terminated, func(i, j int) bool {
		return terminated[i].SectorID < terminated[j].SectorID
	})
	for i, e := range terminated {
		require.Equal(t, minermodel.SectorTerminated, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	recovered := events[minermodel.SectorRecovered]
	sort.Slice(recovered, func(i, j int) bool {
		return recovered[i].SectorID < recovered[j].SectorID
	})
	for i, e := range recovered {
		require.Equal(t, minermodel.SectorRecovered, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	faulted := events[minermodel.SectorFaulted]
	sort.Slice(faulted, func(i, j int) bool {
		return faulted[i].SectorID < faulted[j].SectorID
	})
	for i, e := range faulted {
		require.Equal(t, minermodel.SectorFaulted, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	recovering := events[minermodel.SectorRecovering]
	sort.Slice(recovering, func(i, j int) bool {
		return recovering[i].SectorID < recovering[j].SectorID
	})
	for i, e := range recovering {
		require.Equal(t, minermodel.SectorRecovering, e.Event)
		require.Equal(t, int64(ts.Height()), e.Height)
		require.Equal(t, addr.String(), e.MinerID)
		require.Equal(t, ts.ParentState().String(), e.StateRoot)
		require.EqualValues(t, i, e.SectorID)
	}

	precommit := events[minermodel.PreCommitAdded]
	sort.Slice(precommit, func(i, j int) bool {
		return precommit[i].SectorID < precommit[j].SectorID
	})
	for i, res := range precommit {
		require.Equal(t, addr.String(), res.MinerID)
		require.Equal(t, int64(ts.Height()), res.Height)
		require.Equal(t, ts.ParentState().String(), res.StateRoot)
		require.Equal(t, minermodel.PreCommitAdded, res.Event)
		require.EqualValues(t, fakePrecommitChanges.Added[i].Info.SectorNumber, res.SectorID)
	}
}

func generateFakeSectorOnChainInfo(sectorNumber uint64, dealIDs ...abi.DealID) miner.SectorOnChainInfo {
	return miner.SectorOnChainInfo{
		SectorNumber: abi.SectorNumber(sectorNumber),
		DealIDs:      dealIDs,
		// faked
		SealProof:             0,
		SealedCID:             cid.Undef,
		Activation:            0,
		Expiration:            0,
		DealWeight:            abi.DealWeight{},
		VerifiedDealWeight:    abi.DealWeight{},
		InitialPledge:         abi.TokenAmount{},
		ExpectedDayReward:     abi.TokenAmount{},
		ExpectedStoragePledge: abi.TokenAmount{},
		SectorKeyCID:          nil,
	}
}

func generateFakeSectorOnChainInfos(sectors map[uint64][]abi.DealID) []miner.SectorOnChainInfo {
	var out []miner.SectorOnChainInfo
	for sectorNumber, deals := range sectors {
		out = append(out, generateFakeSectorOnChainInfo(sectorNumber, deals...))
	}
	return out
}

func generateFakeSectorPreCommitInfo(sectorNumber uint64) minertypes.SectorPreCommitInfo {
	return minertypes.SectorPreCommitInfo{
		SealProof:     0,
		SectorNumber:  abi.SectorNumber(sectorNumber),
		SealedCID:     cid.Undef,
		SealRandEpoch: 0,
		DealIDs:       []abi.DealID{},
		Expiration:    0,
	}
}

func generateFakeSectorPreCommitOnChainInfo(sectorNumber uint64) minertypes.SectorPreCommitOnChainInfo {
	return minertypes.SectorPreCommitOnChainInfo{
		Info:             generateFakeSectorPreCommitInfo(sectorNumber),
		PreCommitDeposit: abi.NewTokenAmount(0),
		PreCommitEpoch:   0,
	}
}

func generateFakeSectorPreCommitChanges(add uint64, rm uint64) *miner.PreCommitChanges {
	added := make([]minertypes.SectorPreCommitOnChainInfo, add)
	removed := make([]minertypes.SectorPreCommitOnChainInfo, rm)
	for i := uint64(0); i < add; i++ {
		added[i] = generateFakeSectorPreCommitOnChainInfo(i)
	}
	for i := uint64(0); i < rm; i++ {
		removed[i] = generateFakeSectorPreCommitOnChainInfo(i)
	}

	return &miner.PreCommitChanges{
		Added:   added,
		Removed: removed,
	}
}

func generateFakeSectorStateChanges(removed, recovered, faulted, recovering uint64) *minerex.SectorStateEvents {
	rm := bitfield.New()
	for i := uint64(0); i < removed; i++ {
		rm.Set(i)
	}
	rec := bitfield.New()
	for i := uint64(0); i < recovered; i++ {
		rec.Set(i)
	}
	flt := bitfield.New()
	for i := uint64(0); i < faulted; i++ {
		flt.Set(i)
	}
	recing := bitfield.New()
	for i := uint64(0); i < recovering; i++ {
		recing.Set(i)
	}
	return &minerex.SectorStateEvents{
		Removed:    rm,
		Recovered:  rec,
		Faulted:    flt,
		Recovering: recing,
	}
}

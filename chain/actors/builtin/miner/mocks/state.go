// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	abi "github.com/filecoin-project/go-state-types/abi"
	actors "github.com/filecoin-project/go-state-types/actors"

	adt "github.com/filecoin-project/lotus/chain/actors/adt"

	big "github.com/filecoin-project/go-state-types/big"

	bitfield "github.com/filecoin-project/go-bitfield"

	cid "github.com/ipfs/go-cid"

	dline "github.com/filecoin-project/go-state-types/dline"

	io "io"

	miner "github.com/filecoin-project/lily/chain/actors/builtin/miner"

	mock "github.com/stretchr/testify/mock"

	typegen "github.com/whyrusleeping/cbor-gen"

	v10miner "github.com/filecoin-project/go-state-types/builtin/v10/miner"

	v8miner "github.com/filecoin-project/go-state-types/builtin/v8/miner"

	v9miner "github.com/filecoin-project/go-state-types/builtin/v9/miner"
)

// State is an autogenerated mock type for the State type
type State struct {
	mock.Mock
}

// ActorKey provides a mock function with given fields:
func (_m *State) ActorKey() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// ActorVersion provides a mock function with given fields:
func (_m *State) ActorVersion() actors.Version {
	ret := _m.Called()

	var r0 actors.Version
	if rf, ok := ret.Get(0).(func() actors.Version); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(actors.Version)
	}

	return r0
}

// AvailableBalance provides a mock function with given fields: _a0
func (_m *State) AvailableBalance(_a0 big.Int) (big.Int, error) {
	ret := _m.Called(_a0)

	var r0 big.Int
	if rf, ok := ret.Get(0).(func(big.Int) big.Int); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(big.Int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(big.Int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Code provides a mock function with given fields:
func (_m *State) Code() cid.Cid {
	ret := _m.Called()

	var r0 cid.Cid
	if rf, ok := ret.Get(0).(func() cid.Cid); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(cid.Cid)
	}

	return r0
}

// DeadlineCronActive provides a mock function with given fields:
func (_m *State) DeadlineCronActive() (bool, error) {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeadlineInfo provides a mock function with given fields: epoch
func (_m *State) DeadlineInfo(epoch abi.ChainEpoch) (*dline.Info, error) {
	ret := _m.Called(epoch)

	var r0 *dline.Info
	if rf, ok := ret.Get(0).(func(abi.ChainEpoch) *dline.Info); ok {
		r0 = rf(epoch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dline.Info)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(abi.ChainEpoch) error); ok {
		r1 = rf(epoch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeadlinesChanged provides a mock function with given fields: _a0
func (_m *State) DeadlinesChanged(_a0 miner.State) (bool, error) {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(miner.State) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(miner.State) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeSectorOnChainInfo provides a mock function with given fields: _a0
func (_m *State) DecodeSectorOnChainInfo(_a0 *typegen.Deferred) (v10miner.SectorOnChainInfo, error) {
	ret := _m.Called(_a0)

	var r0 v10miner.SectorOnChainInfo
	if rf, ok := ret.Get(0).(func(*typegen.Deferred) v10miner.SectorOnChainInfo); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(v10miner.SectorOnChainInfo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*typegen.Deferred) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeSectorPreCommitOnChainInfo provides a mock function with given fields: _a0
func (_m *State) DecodeSectorPreCommitOnChainInfo(_a0 *typegen.Deferred) (v9miner.SectorPreCommitOnChainInfo, error) {
	ret := _m.Called(_a0)

	var r0 v9miner.SectorPreCommitOnChainInfo
	if rf, ok := ret.Get(0).(func(*typegen.Deferred) v9miner.SectorPreCommitOnChainInfo); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(v9miner.SectorPreCommitOnChainInfo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*typegen.Deferred) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeSectorPreCommitOnChainInfoToV8 provides a mock function with given fields: _a0
func (_m *State) DecodeSectorPreCommitOnChainInfoToV8(_a0 *typegen.Deferred) (v8miner.SectorPreCommitOnChainInfo, error) {
	ret := _m.Called(_a0)

	var r0 v8miner.SectorPreCommitOnChainInfo
	if rf, ok := ret.Get(0).(func(*typegen.Deferred) v8miner.SectorPreCommitOnChainInfo); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(v8miner.SectorPreCommitOnChainInfo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*typegen.Deferred) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EraseAllUnproven provides a mock function with given fields:
func (_m *State) EraseAllUnproven() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FeeDebt provides a mock function with given fields:
func (_m *State) FeeDebt() (big.Int, error) {
	ret := _m.Called()

	var r0 big.Int
	if rf, ok := ret.Get(0).(func() big.Int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(big.Int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindSector provides a mock function with given fields: _a0
func (_m *State) FindSector(_a0 abi.SectorNumber) (*miner.SectorLocation, error) {
	ret := _m.Called(_a0)

	var r0 *miner.SectorLocation
	if rf, ok := ret.Get(0).(func(abi.SectorNumber) *miner.SectorLocation); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*miner.SectorLocation)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(abi.SectorNumber) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ForEachDeadline provides a mock function with given fields: cb
func (_m *State) ForEachDeadline(cb func(uint64, miner.Deadline) error) error {
	ret := _m.Called(cb)

	var r0 error
	if rf, ok := ret.Get(0).(func(func(uint64, miner.Deadline) error) error); ok {
		r0 = rf(cb)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ForEachPrecommittedSector provides a mock function with given fields: _a0
func (_m *State) ForEachPrecommittedSector(_a0 func(v9miner.SectorPreCommitOnChainInfo) error) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(func(v9miner.SectorPreCommitOnChainInfo) error) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ForEachPrecommittedSectorV8 provides a mock function with given fields: _a0
func (_m *State) ForEachPrecommittedSectorV8(_a0 func(v8miner.SectorPreCommitOnChainInfo) error) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(func(v8miner.SectorPreCommitOnChainInfo) error) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAllocatedSectors provides a mock function with given fields:
func (_m *State) GetAllocatedSectors() (*bitfield.BitField, error) {
	ret := _m.Called()

	var r0 *bitfield.BitField
	if rf, ok := ret.Get(0).(func() *bitfield.BitField); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bitfield.BitField)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPrecommittedSector provides a mock function with given fields: _a0
func (_m *State) GetPrecommittedSector(_a0 abi.SectorNumber) (*v9miner.SectorPreCommitOnChainInfo, error) {
	ret := _m.Called(_a0)

	var r0 *v9miner.SectorPreCommitOnChainInfo
	if rf, ok := ret.Get(0).(func(abi.SectorNumber) *v9miner.SectorPreCommitOnChainInfo); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v9miner.SectorPreCommitOnChainInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(abi.SectorNumber) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetProvingPeriodStart provides a mock function with given fields:
func (_m *State) GetProvingPeriodStart() (abi.ChainEpoch, error) {
	ret := _m.Called()

	var r0 abi.ChainEpoch
	if rf, ok := ret.Get(0).(func() abi.ChainEpoch); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(abi.ChainEpoch)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSector provides a mock function with given fields: _a0
func (_m *State) GetSector(_a0 abi.SectorNumber) (*v10miner.SectorOnChainInfo, error) {
	ret := _m.Called(_a0)

	var r0 *v10miner.SectorOnChainInfo
	if rf, ok := ret.Get(0).(func(abi.SectorNumber) *v10miner.SectorOnChainInfo); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v10miner.SectorOnChainInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(abi.SectorNumber) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSectorExpiration provides a mock function with given fields: _a0
func (_m *State) GetSectorExpiration(_a0 abi.SectorNumber) (*miner.SectorExpiration, error) {
	ret := _m.Called(_a0)

	var r0 *miner.SectorExpiration
	if rf, ok := ret.Get(0).(func(abi.SectorNumber) *miner.SectorExpiration); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*miner.SectorExpiration)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(abi.SectorNumber) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetState provides a mock function with given fields:
func (_m *State) GetState() interface{} {
	ret := _m.Called()

	var r0 interface{}
	if rf, ok := ret.Get(0).(func() interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// Info provides a mock function with given fields:
func (_m *State) Info() (v9miner.MinerInfo, error) {
	ret := _m.Called()

	var r0 v9miner.MinerInfo
	if rf, ok := ret.Get(0).(func() v9miner.MinerInfo); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(v9miner.MinerInfo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsAllocated provides a mock function with given fields: _a0
func (_m *State) IsAllocated(_a0 abi.SectorNumber) (bool, error) {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(abi.SectorNumber) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(abi.SectorNumber) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadDeadline provides a mock function with given fields: idx
func (_m *State) LoadDeadline(idx uint64) (miner.Deadline, error) {
	ret := _m.Called(idx)

	var r0 miner.Deadline
	if rf, ok := ret.Get(0).(func(uint64) miner.Deadline); ok {
		r0 = rf(idx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(miner.Deadline)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(idx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadSectors provides a mock function with given fields: sectorNos
func (_m *State) LoadSectors(sectorNos *bitfield.BitField) ([]*v10miner.SectorOnChainInfo, error) {
	ret := _m.Called(sectorNos)

	var r0 []*v10miner.SectorOnChainInfo
	if rf, ok := ret.Get(0).(func(*bitfield.BitField) []*v10miner.SectorOnChainInfo); ok {
		r0 = rf(sectorNos)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*v10miner.SectorOnChainInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*bitfield.BitField) error); ok {
		r1 = rf(sectorNos)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LockedFunds provides a mock function with given fields:
func (_m *State) LockedFunds() (miner.LockedFunds, error) {
	ret := _m.Called()

	var r0 miner.LockedFunds
	if rf, ok := ret.Get(0).(func() miner.LockedFunds); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(miner.LockedFunds)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MarshalCBOR provides a mock function with given fields: w
func (_m *State) MarshalCBOR(w io.Writer) error {
	ret := _m.Called(w)

	var r0 error
	if rf, ok := ret.Get(0).(func(io.Writer) error); ok {
		r0 = rf(w)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MinerInfoChanged provides a mock function with given fields: _a0
func (_m *State) MinerInfoChanged(_a0 miner.State) (bool, error) {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(miner.State) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(miner.State) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NumDeadlines provides a mock function with given fields:
func (_m *State) NumDeadlines() (uint64, error) {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NumLiveSectors provides a mock function with given fields:
func (_m *State) NumLiveSectors() (uint64, error) {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PrecommitsMap provides a mock function with given fields:
func (_m *State) PrecommitsMap() (adt.Map, error) {
	ret := _m.Called()

	var r0 adt.Map
	if rf, ok := ret.Get(0).(func() adt.Map); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(adt.Map)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PrecommitsMapBitWidth provides a mock function with given fields:
func (_m *State) PrecommitsMapBitWidth() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// PrecommitsMapHashFunction provides a mock function with given fields:
func (_m *State) PrecommitsMapHashFunction() func([]byte) []byte {
	ret := _m.Called()

	var r0 func([]byte) []byte
	if rf, ok := ret.Get(0).(func() func([]byte) []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(func([]byte) []byte)
		}
	}

	return r0
}

// SectorsAmtBitwidth provides a mock function with given fields:
func (_m *State) SectorsAmtBitwidth() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// SectorsArray provides a mock function with given fields:
func (_m *State) SectorsArray() (adt.Array, error) {
	ret := _m.Called()

	var r0 adt.Array
	if rf, ok := ret.Get(0).(func() adt.Array); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(adt.Array)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UnallocatedSectorNumbers provides a mock function with given fields: count
func (_m *State) UnallocatedSectorNumbers(count int) ([]abi.SectorNumber, error) {
	ret := _m.Called(count)

	var r0 []abi.SectorNumber
	if rf, ok := ret.Get(0).(func(int) []abi.SectorNumber); ok {
		r0 = rf(count)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]abi.SectorNumber)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(count)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VestedFunds provides a mock function with given fields: _a0
func (_m *State) VestedFunds(_a0 abi.ChainEpoch) (big.Int, error) {
	ret := _m.Called(_a0)

	var r0 big.Int
	if rf, ok := ret.Get(0).(func(abi.ChainEpoch) big.Int); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(big.Int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(abi.ChainEpoch) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewState interface {
	mock.TestingT
	Cleanup(func())
}

// NewState creates a new instance of State. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewState(t mockConstructorTestingTNewState) *State {
	mock := &State{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

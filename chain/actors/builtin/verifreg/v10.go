// Code generated by: `make actors-gen`. DO NOT EDIT.
package verifreg

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors/adt"

	"crypto/sha256"

	builtin10 "github.com/filecoin-project/go-state-types/builtin"
	adt10 "github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	verifreg10 "github.com/filecoin-project/go-state-types/builtin/v10/verifreg"

	"github.com/filecoin-project/go-state-types/big"

	verifreg9 "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"

	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lotus/chain/actors"
)

var _ State = (*state10)(nil)

func load10(store adt.Store, root cid.Cid) (State, error) {
	out := state10{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state10 struct {
	verifreg10.State
	store adt.Store
}

func (s *state10) ActorKey() string {
	return manifest.VerifregKey
}

func (s *state10) ActorVersion() actorstypes.Version {
	return actorstypes.Version10
}

func (s *state10) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

func (s *state10) VerifiedClientsMapBitWidth() int {

	return builtin10.DefaultHamtBitwidth

}

func (s *state10) VerifiedClientsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state10) VerifiedClientsMap() (adt.Map, error) {

	return nil, fmt.Errorf("unsupported in actors v10")

}

func (s *state10) VerifiersMap() (adt.Map, error) {
	return adt10.AsMap(s.store, s.Verifiers, builtin10.DefaultHamtBitwidth)
}

func (s *state10) VerifiersMapBitWidth() int {

	return builtin10.DefaultHamtBitwidth

}

func (s *state10) VerifiersMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state10) RootKey() (address.Address, error) {
	return s.State.RootKey, nil
}

func (s *state10) VerifiedClientDataCap(addr address.Address) (bool, abi.StoragePower, error) {

	return false, big.Zero(), fmt.Errorf("unsupported in actors v10")

}

func (s *state10) VerifierDataCap(addr address.Address) (bool, abi.StoragePower, error) {
	return getDataCap(s.store, actorstypes.Version10, s.VerifiersMap, addr)
}

func (s *state10) RemoveDataCapProposalID(verifier address.Address, client address.Address) (bool, uint64, error) {
	return getRemoveDataCapProposalID(s.store, actorstypes.Version10, s.removeDataCapProposalIDs, verifier, client)
}

func (s *state10) ForEachVerifier(cb func(addr address.Address, dcap abi.StoragePower) error) error {
	return forEachCap(s.store, actorstypes.Version10, s.VerifiersMap, cb)
}

func (s *state10) ForEachClient(cb func(addr address.Address, dcap abi.StoragePower) error) error {

	return fmt.Errorf("unsupported in actors v10")

}

func (s *state10) removeDataCapProposalIDs() (adt.Map, error) {
	return adt10.AsMap(s.store, s.RemoveDataCapProposalIDs, builtin10.DefaultHamtBitwidth)
}

func (s *state10) GetState() interface{} {
	return &s.State
}

func (s *state10) GetAllocation(clientIdAddr address.Address, allocationId verifreg9.AllocationId) (*Allocation, bool, error) {

	alloc, ok, err := s.FindAllocation(s.store, clientIdAddr, verifreg10.AllocationId(allocationId))
	return (*Allocation)(alloc), ok, err
}

func (s *state10) GetAllocations(clientIdAddr address.Address) (map[AllocationId]Allocation, error) {

	v10Map, err := s.LoadAllocationsToMap(s.store, clientIdAddr)

	retMap := make(map[AllocationId]Allocation, len(v10Map))
	for k, v := range v10Map {
		retMap[AllocationId(k)] = Allocation(v)
	}

	return retMap, err

}

func (s *state10) GetClaim(providerIdAddr address.Address, claimId verifreg9.ClaimId) (*Claim, bool, error) {

	claim, ok, err := s.FindClaim(s.store, providerIdAddr, verifreg10.ClaimId(claimId))
	return (*Claim)(claim), ok, err

}

func (s *state10) GetClaims(providerIdAddr address.Address) (map[ClaimId]Claim, error) {

	v10Map, err := s.LoadClaimsToMap(s.store, providerIdAddr)

	retMap := make(map[ClaimId]Claim, len(v10Map))
	for k, v := range v10Map {
		retMap[ClaimId(k)] = Claim(v)
	}

	return retMap, err

}
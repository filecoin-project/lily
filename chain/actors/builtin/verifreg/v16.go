// Code generated by: `make actors-gen`. DO NOT EDIT.
package verifreg

import (
	"crypto/sha256"
	"fmt"

	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/big"
	builtin16 "github.com/filecoin-project/go-state-types/builtin"
	adt16 "github.com/filecoin-project/go-state-types/builtin/v16/util/adt"
	verifreg16 "github.com/filecoin-project/go-state-types/builtin/v16/verifreg"
	verifreg9 "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lily/chain/actors/adt"

	"github.com/filecoin-project/lotus/chain/actors"
)

var _ State = (*state16)(nil)

func load16(store adt.Store, root cid.Cid) (State, error) {
	out := state16{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state16 struct {
	verifreg16.State
	store adt.Store
}

func (s *state16) ActorKey() string {
	return manifest.VerifregKey
}

func (s *state16) ActorVersion() actorstypes.Version {
	return actorstypes.Version16
}

func (s *state16) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

func (s *state16) VerifiedClientsMapBitWidth() int {

	return builtin16.DefaultHamtBitwidth

}

func (s *state16) VerifiedClientsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state16) VerifiedClientsMap() (adt.Map, error) {

	return nil, fmt.Errorf("unsupported in actors v16")

}

func (s *state16) VerifiersMap() (adt.Map, error) {
	return adt16.AsMap(s.store, s.Verifiers, builtin16.DefaultHamtBitwidth)
}

func (s *state16) VerifiersMapBitWidth() int {

	return builtin16.DefaultHamtBitwidth

}

func (s *state16) VerifiersMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state16) RootKey() (address.Address, error) {
	return s.State.RootKey, nil
}

func (s *state16) VerifiedClientDataCap(addr address.Address) (bool, abi.StoragePower, error) {

	return false, big.Zero(), fmt.Errorf("unsupported in actors v16")

}

func (s *state16) VerifierDataCap(addr address.Address) (bool, abi.StoragePower, error) {
	return getDataCap(s.store, actorstypes.Version16, s.VerifiersMap, addr)
}

func (s *state16) RemoveDataCapProposalID(verifier address.Address, client address.Address) (bool, uint64, error) {
	return getRemoveDataCapProposalID(s.store, actorstypes.Version16, s.removeDataCapProposalIDs, verifier, client)
}

func (s *state16) ForEachVerifier(cb func(addr address.Address, dcap abi.StoragePower) error) error {
	return forEachCap(s.store, actorstypes.Version16, s.VerifiersMap, cb)
}

func (s *state16) ForEachClient(cb func(addr address.Address, dcap abi.StoragePower) error) error {

	return fmt.Errorf("unsupported in actors v16")

}

func (s *state16) removeDataCapProposalIDs() (adt.Map, error) {
	return adt16.AsMap(s.store, s.RemoveDataCapProposalIDs, builtin16.DefaultHamtBitwidth)
}

func (s *state16) GetState() interface{} {
	return &s.State
}

func (s *state16) GetAllocation(clientIdAddr address.Address, allocationId verifreg9.AllocationId) (*Allocation, bool, error) {

	alloc, ok, err := s.FindAllocation(s.store, clientIdAddr, verifreg16.AllocationId(allocationId))
	return (*Allocation)(alloc), ok, err
}

func (s *state16) GetAllocations(clientIdAddr address.Address) (map[AllocationId]Allocation, error) {

	v16Map, err := s.LoadAllocationsToMap(s.store, clientIdAddr)

	retMap := make(map[AllocationId]Allocation, len(v16Map))
	for k, v := range v16Map {
		retMap[AllocationId(k)] = Allocation(v)
	}

	return retMap, err

}

func (s *state16) GetClaim(providerIdAddr address.Address, claimId verifreg9.ClaimId) (*Claim, bool, error) {

	claim, ok, err := s.FindClaim(s.store, providerIdAddr, verifreg16.ClaimId(claimId))
	return (*Claim)(claim), ok, err

}

func (s *state16) GetClaims(providerIdAddr address.Address) (map[ClaimId]Claim, error) {

	v16Map, err := s.LoadClaimsToMap(s.store, providerIdAddr)

	retMap := make(map[ClaimId]Claim, len(v16Map))
	for k, v := range v16Map {
		retMap[ClaimId(k)] = Claim(v)
	}

	return retMap, err

}

func (s *state16) ClaimsMap() (adt.Map, error) {

	return adt16.AsMap(s.store, s.Claims, builtin16.DefaultHamtBitwidth)

}

// TODO this could return an error since not all versions have a claims map
func (s *state16) ClaimsMapBitWidth() int {

	return builtin16.DefaultHamtBitwidth

}

// TODO this could return an error since not all versions have a claims map
func (s *state16) ClaimsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state16) ClaimMapForProvider(providerIdAddr address.Address) (adt.Map, error) {

	innerHamtCid, err := s.getInnerHamtCid(s.store, abi.IdAddrKey(providerIdAddr), s.Claims, builtin16.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}
	return adt16.AsMap(s.store, innerHamtCid, builtin16.DefaultHamtBitwidth)

}

func (s *state16) getInnerHamtCid(store adt.Store, key abi.Keyer, mapCid cid.Cid, bitwidth int) (cid.Cid, error) {

	actorToHamtMap, err := adt16.AsMap(store, mapCid, bitwidth)
	if err != nil {
		return cid.Undef, fmt.Errorf("couldn't get outer map: %x", err)
	}

	var innerHamtCid cbg.CborCid
	if found, err := actorToHamtMap.Get(key, &innerHamtCid); err != nil {
		return cid.Undef, fmt.Errorf("looking up key: %s: %w", key, err)
	} else if !found {
		return cid.Undef, fmt.Errorf("did not find key: %s", key)
	}

	return cid.Cid(innerHamtCid), nil

}

// Code generated by: `make actors-gen`. DO NOT EDIT.
package verifreg

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors/adt"

	"crypto/sha256"

	builtin9 "github.com/filecoin-project/go-state-types/builtin"
	adt9 "github.com/filecoin-project/go-state-types/builtin/v9/util/adt"
	verifreg9 "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"

	"github.com/filecoin-project/go-state-types/big"

	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lotus/chain/actors"

	cbg "github.com/whyrusleeping/cbor-gen"
)

var _ State = (*state9)(nil)

func load9(store adt.Store, root cid.Cid) (State, error) {
	out := state9{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state9 struct {
	verifreg9.State
	store adt.Store
}

func (s *state9) ActorKey() string {
	return manifest.VerifregKey
}

func (s *state9) ActorVersion() actorstypes.Version {
	return actorstypes.Version9
}

func (s *state9) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

func (s *state9) VerifiedClientsMapBitWidth() int {

	return builtin9.DefaultHamtBitwidth

}

func (s *state9) VerifiedClientsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state9) VerifiedClientsMap() (adt.Map, error) {

	return nil, fmt.Errorf("unsupported in actors v9")

}

func (s *state9) VerifiersMap() (adt.Map, error) {
	return adt9.AsMap(s.store, s.Verifiers, builtin9.DefaultHamtBitwidth)
}

func (s *state9) VerifiersMapBitWidth() int {

	return builtin9.DefaultHamtBitwidth

}

func (s *state9) VerifiersMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state9) RootKey() (address.Address, error) {
	return s.State.RootKey, nil
}

func (s *state9) VerifiedClientDataCap(addr address.Address) (bool, abi.StoragePower, error) {

	return false, big.Zero(), fmt.Errorf("unsupported in actors v9")

}

func (s *state9) VerifierDataCap(addr address.Address) (bool, abi.StoragePower, error) {
	return getDataCap(s.store, actorstypes.Version9, s.VerifiersMap, addr)
}

func (s *state9) RemoveDataCapProposalID(verifier address.Address, client address.Address) (bool, uint64, error) {
	return getRemoveDataCapProposalID(s.store, actorstypes.Version9, s.removeDataCapProposalIDs, verifier, client)
}

func (s *state9) ForEachVerifier(cb func(addr address.Address, dcap abi.StoragePower) error) error {
	return forEachCap(s.store, actorstypes.Version9, s.VerifiersMap, cb)
}

func (s *state9) ForEachClient(cb func(addr address.Address, dcap abi.StoragePower) error) error {

	return fmt.Errorf("unsupported in actors v9")

}

func (s *state9) removeDataCapProposalIDs() (adt.Map, error) {
	return adt9.AsMap(s.store, s.RemoveDataCapProposalIDs, builtin9.DefaultHamtBitwidth)
}

func (s *state9) GetState() interface{} {
	return &s.State
}

func (s *state9) GetAllocation(clientIdAddr address.Address, allocationId verifreg9.AllocationId) (*Allocation, bool, error) {

	alloc, ok, err := s.FindAllocation(s.store, clientIdAddr, verifreg9.AllocationId(allocationId))
	return (*Allocation)(alloc), ok, err
}

func (s *state9) GetAllocations(clientIdAddr address.Address) (map[AllocationId]Allocation, error) {

	v9Map, err := s.LoadAllocationsToMap(s.store, clientIdAddr)

	retMap := make(map[AllocationId]Allocation, len(v9Map))
	for k, v := range v9Map {
		retMap[AllocationId(k)] = Allocation(v)
	}

	return retMap, err

}

func (s *state9) GetClaim(providerIdAddr address.Address, claimId verifreg9.ClaimId) (*Claim, bool, error) {

	claim, ok, err := s.FindClaim(s.store, providerIdAddr, verifreg9.ClaimId(claimId))
	return (*Claim)(claim), ok, err

}

func (s *state9) GetClaims(providerIdAddr address.Address) (map[ClaimId]Claim, error) {

	v9Map, err := s.LoadClaimsToMap(s.store, providerIdAddr)

	retMap := make(map[ClaimId]Claim, len(v9Map))
	for k, v := range v9Map {
		retMap[ClaimId(k)] = Claim(v)
	}

	return retMap, err

}

func (s *state9) ClaimsMap() (adt.Map, error) {

	return adt9.AsMap(s.store, s.Claims, builtin9.DefaultHamtBitwidth)

}

// TODO this could return an error since not all versions have a claims map
func (s *state9) ClaimsMapBitWidth() int {

	return builtin9.DefaultHamtBitwidth

}

// TODO this could return an error since not all versions have a claims map
func (s *state9) ClaimsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state9) ClaimMapForProvider(providerIdAddr address.Address) (adt.Map, error) {

	innerHamtCid, err := s.getInnerHamtCid(s.store, abi.IdAddrKey(providerIdAddr), s.Claims, builtin9.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}
	return adt9.AsMap(s.store, innerHamtCid, builtin9.DefaultHamtBitwidth)

}

func (s *state9) getInnerHamtCid(store adt.Store, key abi.Keyer, mapCid cid.Cid, bitwidth int) (cid.Cid, error) {

	actorToHamtMap, err := adt9.AsMap(store, mapCid, bitwidth)
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

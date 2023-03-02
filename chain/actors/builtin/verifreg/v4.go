// Code generated by: `make actors-gen`. DO NOT EDIT.
package verifreg

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors/adt"

	"crypto/sha256"

	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"

	verifreg4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/verifreg"
	adt4 "github.com/filecoin-project/specs-actors/v4/actors/util/adt"

	verifreg9 "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"

	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lotus/chain/actors"
)

var _ State = (*state4)(nil)

func load4(store adt.Store, root cid.Cid) (State, error) {
	out := state4{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state4 struct {
	verifreg4.State
	store adt.Store
}

func (s *state4) ActorKey() string {
	return manifest.VerifregKey
}

func (s *state4) ActorVersion() actorstypes.Version {
	return actorstypes.Version4
}

func (s *state4) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

func (s *state4) VerifiedClientsMapBitWidth() int {

	return builtin4.DefaultHamtBitwidth

}

func (s *state4) VerifiedClientsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state4) VerifiedClientsMap() (adt.Map, error) {

	return adt4.AsMap(s.store, s.VerifiedClients, builtin4.DefaultHamtBitwidth)

}

func (s *state4) VerifiersMap() (adt.Map, error) {
	return adt4.AsMap(s.store, s.Verifiers, builtin4.DefaultHamtBitwidth)
}

func (s *state4) VerifiersMapBitWidth() int {

	return builtin4.DefaultHamtBitwidth

}

func (s *state4) VerifiersMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state4) RootKey() (address.Address, error) {
	return s.State.RootKey, nil
}

func (s *state4) VerifiedClientDataCap(addr address.Address) (bool, abi.StoragePower, error) {

	return getDataCap(s.store, actorstypes.Version4, s.VerifiedClientsMap, addr)

}

func (s *state4) VerifierDataCap(addr address.Address) (bool, abi.StoragePower, error) {
	return getDataCap(s.store, actorstypes.Version4, s.VerifiersMap, addr)
}

func (s *state4) RemoveDataCapProposalID(verifier address.Address, client address.Address) (bool, uint64, error) {
	return getRemoveDataCapProposalID(s.store, actorstypes.Version4, s.removeDataCapProposalIDs, verifier, client)
}

func (s *state4) ForEachVerifier(cb func(addr address.Address, dcap abi.StoragePower) error) error {
	return forEachCap(s.store, actorstypes.Version4, s.VerifiersMap, cb)
}

func (s *state4) ForEachClient(cb func(addr address.Address, dcap abi.StoragePower) error) error {

	return forEachCap(s.store, actorstypes.Version4, s.VerifiedClientsMap, cb)

}

func (s *state4) removeDataCapProposalIDs() (adt.Map, error) {
	return nil, nil

}

func (s *state4) GetState() interface{} {
	return &s.State
}

func (s *state4) GetAllocation(clientIdAddr address.Address, allocationId verifreg9.AllocationId) (*Allocation, bool, error) {

	return nil, false, fmt.Errorf("unsupported in actors v4")

}

func (s *state4) GetAllocations(clientIdAddr address.Address) (map[AllocationId]Allocation, error) {

	return nil, fmt.Errorf("unsupported in actors v4")

}

func (s *state4) GetClaim(providerIdAddr address.Address, claimId verifreg9.ClaimId) (*Claim, bool, error) {

	return nil, false, fmt.Errorf("unsupported in actors v4")

}

func (s *state4) GetClaims(providerIdAddr address.Address) (map[ClaimId]Claim, error) {

	return nil, fmt.Errorf("unsupported in actors v4")

}

func (s *state4) ClaimsMap() (adt.Map, error) {

	return nil, fmt.Errorf("unsupported in actors v4")

}

// TODO this could return an error since not all versions have a claims map
func (s *state4) ClaimsMapBitWidth() int {

	return builtin4.DefaultHamtBitwidth

}

// TODO this could return an error since not all versions have a claims map
func (s *state4) ClaimsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state4) ClaimMapForProvider(providerIdAddr address.Address) (adt.Map, error) {

	return nil, fmt.Errorf("unsupported in actors v4")

}

func (s *state4) getInnerHamtCid(store adt.Store, key abi.Keyer, mapCid cid.Cid, bitwidth int) (cid.Cid, error) {

	return cid.Undef, fmt.Errorf("unsupported in actors v4")

}

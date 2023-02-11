package datacap

import (
	"crypto/sha256"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lotus/chain/actors/adt"

	"github.com/filecoin-project/lily/chain/actors"

	datacap9 "github.com/filecoin-project/go-state-types/builtin/v9/datacap"
	adt9 "github.com/filecoin-project/go-state-types/builtin/v9/util/adt"
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

func make9(store adt.Store, governor address.Address, bitwidth uint64) (State, error) {
	out := state9{store: store}
	s, err := datacap9.ConstructState(store, governor, bitwidth)
	if err != nil {
		return nil, err
	}

	out.State = *s

	return &out, nil
}

type state9 struct {
	datacap9.State
	store adt.Store
}

func (s *state9) Governor() (address.Address, error) {
	return s.State.Governor, nil
}

func (s *state9) GetState() interface{} {
	return &s.State
}

func (s *state9) ForEachClient(cb func(addr address.Address, dcap abi.StoragePower) error) error {
	return forEachClient(s.store, actors.Version9, s.VerifiedClients, cb)
}

func (s *state9) VerifiedClients() (adt.Map, error) {
	return adt9.AsMap(s.store, s.Token.Balances, int(s.Token.HamtBitWidth))
}

func (s *state9) VerifiedClientDataCap(addr address.Address) (bool, abi.StoragePower, error) {
	return getDataCap(s.store, actors.Version9, s.VerifiedClients, addr)
}

func (s *state9) VerifiedClientsMapBitWidth() int {
	return int(s.Token.HamtBitWidth)
}

func (s *state9) VerifiedClientsMapHashFunction() func(input []byte) []byte {
	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}
}

func (s *state9) AllowanceMap() (adt.Map, error) {
	return adt9.AsMap(s.store, s.Token.Allowances, int(s.Token.HamtBitWidth))
}

func (s *state9) AllowanceMapBitWidth() int {
	return int(s.Token.HamtBitWidth)
}

func (s *state9) AllowanceMapHashFunction() func(input []byte) []byte {
	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}
}

func (s *state9) AllowanceMapForOwner(owner address.Address) (adt.Map, error) {
	innerHamtCid, err := s.getInnerHamtCid(s.store, abi.IdAddrKey(owner), s.Token.Allowances, int(s.Token.HamtBitWidth))
	if err != nil {
		return nil, err
	}
	return adt9.AsMap(s.store, innerHamtCid, int(s.Token.HamtBitWidth))
}

func (s *state9) ActorKey() string {
	return actors.DatacapKey
}

func (s *state9) ActorVersion() actors.Version {
	return actors.Version9
}

func (s *state9) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
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

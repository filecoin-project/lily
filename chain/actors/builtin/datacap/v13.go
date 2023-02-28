package datacap

import (
	"crypto/sha256"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"

	datacap13 "github.com/filecoin-project/go-state-types/builtin/v13/datacap"
	adt13 "github.com/filecoin-project/go-state-types/builtin/v13/util/adt"
)

var _ State = (*state13)(nil)

func load13(store adt.Store, root cid.Cid) (State, error) {
	out := state13{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func make13(store adt.Store, governor address.Address, bitwidth uint64) (State, error) {
	out := state13{store: store}
	s, err := datacap13.ConstructState(store, governor, bitwidth)
	if err != nil {
		return nil, err
	}

	out.State = *s

	return &out, nil
}

type state13 struct {
	datacap13.State
	store adt.Store
}

func (s *state13) Governor() (address.Address, error) {
	return s.State.Governor, nil
}

func (s *state13) GetState() interface{} {
	return &s.State
}

func (s *state13) ForEachClient(cb func(addr address.Address, dcap abi.StoragePower) error) error {
	return forEachClient(s.store, actors.Version13, s.VerifiedClients, cb)
}

func (s *state13) VerifiedClients() (adt.Map, error) {
	return adt13.AsMap(s.store, s.Token.Balances, int(s.Token.HamtBitWidth))
}

func (s *state13) VerifiedClientDataCap(addr address.Address) (bool, abi.StoragePower, error) {
	return getDataCap(s.store, actors.Version13, s.VerifiedClients, addr)
}

func (s *state13) VerifiedClientsMapBitWidth() int {
	return int(s.Token.HamtBitWidth)
}

func (s *state13) VerifiedClientsMapHashFunction() func(input []byte) []byte {
	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}
}

func (s *state13) ActorKey() string {
	return actors.DatacapKey
}

func (s *state13) ActorVersion() actors.Version {
	return actors.Version13
}

func (s *state13) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

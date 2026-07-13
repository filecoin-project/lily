package datacap

import (
	"crypto/sha256"
	"fmt"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	datacap19 "github.com/filecoin-project/go-state-types/builtin/v19/datacap"
	adt19 "github.com/filecoin-project/go-state-types/builtin/v19/util/adt"
	"github.com/filecoin-project/go-state-types/manifest"

	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
)

var _ State = (*state19)(nil)

func load19(store adt.Store, root cid.Cid) (State, error) {
	out := state19{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func make19(store adt.Store, governor address.Address, bitwidth uint64) (State, error) {
	out := state19{store: store}
	s, err := datacap19.ConstructState(store, governor, bitwidth)
	if err != nil {
		return nil, err
	}

	out.State = *s

	return &out, nil
}

type state19 struct {
	datacap19.State
	store adt.Store
}

func (s *state19) Governor() (address.Address, error) {
	return s.State.Governor, nil
}

func (s *state19) GetState() interface{} {
	return &s.State
}

func (s *state19) ForEachClient(cb func(addr address.Address, dcap abi.StoragePower) error) error {
	return forEachClient(s.store, actorstypes.Version19, s.VerifiedClients, cb)
}

func (s *state19) VerifiedClients() (adt.Map, error) {
	return adt19.AsMap(s.store, s.Token.Balances, int(s.Token.HamtBitWidth))
}

func (s *state19) VerifiedClientDataCap(addr address.Address) (bool, abi.StoragePower, error) {
	return getDataCap(s.store, actorstypes.Version19, s.VerifiedClients, addr)
}

func (s *state19) VerifiedClientsMapBitWidth() int {
	return int(s.Token.HamtBitWidth)
}

func (s *state19) VerifiedClientsMapHashFunction() func(input []byte) []byte {
	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}
}

func (s *state19) ActorKey() string {
	return manifest.DatacapKey
}

func (s *state19) ActorVersion() actorstypes.Version {
	return actorstypes.Version19
}

func (s *state19) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

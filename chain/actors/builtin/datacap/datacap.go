package datacap

import (
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	builtin13 "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors"
	lotusactors "github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/types"
)

var (
	Address = builtin13.DatacapActorAddr
	Methods = builtin13.MethodsDatacap
)

func Load(store adt.Store, act *types.Actor) (State, error) {
	if name, av, ok := lotusactors.GetActorMetaByCode(act.Code); ok {
		if name != actors.DatacapKey {
			return nil, xerrors.Errorf("actor code is not datacap: %s", name)
		}

		switch av {

		case actorstypes.Version9:
			return load9(store, act.Head)

		case actorstypes.Version10:
			return load10(store, act.Head)

		case actorstypes.Version11:
			return load11(store, act.Head)

		case actorstypes.Version12:
			return load12(store, act.Head)

		case actorstypes.Version13:
			return load13(store, act.Head)

		}
	}

	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

func MakeState(store adt.Store, av actorstypes.Version, governor address.Address, bitwidth uint64) (State, error) {
	switch av {

	case actorstypes.Version9:
		return make9(store, governor, bitwidth)

	case actorstypes.Version10:
		return make10(store, governor, bitwidth)

	case actorstypes.Version11:
		return make11(store, governor, bitwidth)

	case actorstypes.Version12:
		return make12(store, governor, bitwidth)

	case actorstypes.Version13:
		return make13(store, governor, bitwidth)

	default:
		return nil, xerrors.Errorf("datacap actor only valid for actors v9 and above, got %d", av)
	}
}

type State interface {
	cbor.Marshaler

	Code() cid.Cid
	ActorKey() string
	ActorVersion() actors.Version

	ForEachClient(func(addr address.Address, dcap abi.StoragePower) error) error
	VerifiedClientDataCap(address.Address) (bool, abi.StoragePower, error)
	Governor() (address.Address, error)
	GetState() interface{}

	VerifiedClients() (adt.Map, error)
	VerifiedClientsMapBitWidth() int
	VerifiedClientsMapHashFunction() func(input []byte) []byte
}

func AllCodes() []cid.Cid {
	return []cid.Cid{
		(&state9{}).Code(),
		(&state10{}).Code(),
		(&state11{}).Code(),
		(&state12{}).Code(),
		(&state13{}).Code(),
	}
}

func VersionCodes() map[actors.Version]cid.Cid {
	return map[actors.Version]cid.Cid{
		actors.Version9:  (&state9{}).Code(),
		actors.Version10: (&state10{}).Code(),
		actors.Version11: (&state11{}).Code(),
		actors.Version12: (&state12{}).Code(),
		actors.Version13: (&state13{}).Code(),
	}
}

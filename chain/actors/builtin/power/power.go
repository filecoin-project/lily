// Code generated by: `make actors-gen`. DO NOT EDIT.
package power

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	lotusactors "github.com/filecoin-project/lotus/chain/actors"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"

	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"

	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"

	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"

	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"

	builtin10 "github.com/filecoin-project/go-state-types/builtin"

	"github.com/filecoin-project/lily/chain/actors"
)

var (
	Address = builtin10.StoragePowerActorAddr
	Methods = builtin10.MethodsPower
)

func Load(store adt.Store, act *types.Actor) (State, error) {
	if name, av, ok := lotusactors.GetActorMetaByCode(act.Code); ok {
		if name != actors.PowerKey {
			return nil, fmt.Errorf("actor code is not power: %s", name)
		}

		switch actors.Version(av) {

		case actors.Version8:
			return load8(store, act.Head)

		case actors.Version9:
			return load9(store, act.Head)

		case actors.Version10:
			return load10(store, act.Head)

		}
	}

	switch act.Code {

	case builtin0.StoragePowerActorCodeID:
		return load0(store, act.Head)

	case builtin2.StoragePowerActorCodeID:
		return load2(store, act.Head)

	case builtin3.StoragePowerActorCodeID:
		return load3(store, act.Head)

	case builtin4.StoragePowerActorCodeID:
		return load4(store, act.Head)

	case builtin5.StoragePowerActorCodeID:
		return load5(store, act.Head)

	case builtin6.StoragePowerActorCodeID:
		return load6(store, act.Head)

	case builtin7.StoragePowerActorCodeID:
		return load7(store, act.Head)

	}

	return nil, fmt.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	Code() cid.Cid
	ActorKey() string
	ActorVersion() actors.Version

	TotalLocked() (abi.TokenAmount, error)
	TotalPower() (Claim, error)
	TotalCommitted() (Claim, error)
	TotalPowerSmoothed() (builtin.FilterEstimate, error)

	// MinerCounts returns the number of miners. Participating is the number
	// with power above the minimum miner threshold.
	MinerCounts() (participating, total uint64, err error)
	MinerPower(address.Address) (Claim, bool, error)
	MinerNominalPowerMeetsConsensusMinimum(address.Address) (bool, error)
	ListAllMiners() ([]address.Address, error)
	ForEachClaim(func(miner address.Address, claim Claim) error) error
	ClaimsChanged(State) (bool, error)

	// Diff helpers. Used by Diff* functions internally.
	ClaimsMap() (adt.Map, error)
	ClaimsMapBitWidth() int
	ClaimsMapHashFunction() func(input []byte) []byte
	decodeClaim(*cbg.Deferred) (Claim, error)
}

type Claim struct {
	// Sum of raw byte power for a miner's sectors.
	RawBytePower abi.StoragePower

	// Sum of quality adjusted power for a miner's sectors.
	QualityAdjPower abi.StoragePower
}

func AddClaims(a Claim, b Claim) Claim {
	return Claim{
		RawBytePower:    big.Add(a.RawBytePower, b.RawBytePower),
		QualityAdjPower: big.Add(a.QualityAdjPower, b.QualityAdjPower),
	}
}

func AllCodes() []cid.Cid {
	return []cid.Cid{
		(&state0{}).Code(),
		(&state2{}).Code(),
		(&state3{}).Code(),
		(&state4{}).Code(),
		(&state5{}).Code(),
		(&state6{}).Code(),
		(&state7{}).Code(),
		(&state8{}).Code(),
		(&state9{}).Code(),
		(&state10{}).Code(),
	}
}

func VersionCodes() map[actors.Version]cid.Cid {
	return map[actors.Version]cid.Cid{
		actors.Version0:  (&state0{}).Code(),
		actors.Version2:  (&state2{}).Code(),
		actors.Version3:  (&state3{}).Code(),
		actors.Version4:  (&state4{}).Code(),
		actors.Version5:  (&state5{}).Code(),
		actors.Version6:  (&state6{}).Code(),
		actors.Version7:  (&state7{}).Code(),
		actors.Version8:  (&state8{}).Code(),
		actors.Version9:  (&state9{}).Code(),
		actors.Version10: (&state10{}).Code(),
	}
}

// Code generated by: `make actors-gen`. DO NOT EDIT.
package power

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	builtin13 "github.com/filecoin-project/go-state-types/builtin"
	power13 "github.com/filecoin-project/go-state-types/builtin/v13/power"
	adt13 "github.com/filecoin-project/go-state-types/builtin/v13/util/adt"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin"

	"github.com/filecoin-project/lotus/chain/actors"
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

type state13 struct {
	power13.State
	store adt.Store
}

func (s *state13) TotalLocked() (abi.TokenAmount, error) {
	return s.TotalPledgeCollateral, nil
}

func (s *state13) TotalPower() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalRawBytePower,
		QualityAdjPower: s.TotalQualityAdjPower,
	}, nil
}

// Committed power to the network. Includes miners below the minimum threshold.
func (s *state13) TotalCommitted() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalBytesCommitted,
		QualityAdjPower: s.TotalQABytesCommitted,
	}, nil
}

func (s *state13) MinerPower(addr address.Address) (Claim, bool, error) {
	claims, err := s.ClaimsMap()
	if err != nil {
		return Claim{}, false, err
	}
	var claim power13.Claim
	ok, err := claims.Get(abi.AddrKey(addr), &claim)
	if err != nil {
		return Claim{}, false, err
	}
	return Claim{
		RawBytePower:    claim.RawBytePower,
		QualityAdjPower: claim.QualityAdjPower,
	}, ok, nil
}

func (s *state13) MinerNominalPowerMeetsConsensusMinimum(a address.Address) (bool, error) {
	return s.State.MinerNominalPowerMeetsConsensusMinimum(s.store, a)
}

func (s *state13) TotalPowerSmoothed() (builtin.FilterEstimate, error) {
	return builtin.FilterEstimate(s.State.ThisEpochQAPowerSmoothed), nil
}

func (s *state13) MinerCounts() (uint64, uint64, error) {
	return uint64(s.State.MinerAboveMinPowerCount), uint64(s.State.MinerCount), nil
}

func (s *state13) RampStartEpoch() int64 {
	return 0
}
func (s *state13) RampDurationEpochs() uint64 {
	return 0
}

func (s *state13) ListAllMiners() ([]address.Address, error) {
	claims, err := s.ClaimsMap()
	if err != nil {
		return nil, err
	}

	var miners []address.Address
	err = claims.ForEach(nil, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		miners = append(miners, a)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return miners, nil
}

func (s *state13) ForEachClaim(cb func(miner address.Address, claim Claim) error) error {
	claims, err := s.ClaimsMap()
	if err != nil {
		return err
	}

	var claim power13.Claim
	return claims.ForEach(&claim, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		return cb(a, Claim{
			RawBytePower:    claim.RawBytePower,
			QualityAdjPower: claim.QualityAdjPower,
		})
	})
}

func (s *state13) ClaimsChanged(other State) (bool, error) {
	other13, ok := other.(*state13)
	if !ok {
		// treat an upgrade as a change, always
		return true, nil
	}
	return !s.State.Claims.Equals(other13.State.Claims), nil
}

func (s *state13) ClaimsMap() (adt.Map, error) {
	return adt13.AsMap(s.store, s.Claims, builtin13.DefaultHamtBitwidth)
}

func (s *state13) ClaimsMapBitWidth() int {

	return builtin13.DefaultHamtBitwidth

}

func (s *state13) ClaimsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state13) decodeClaim(val *cbg.Deferred) (Claim, error) {
	var ci power13.Claim
	if err := ci.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return Claim{}, err
	}
	return fromV13Claim(ci), nil
}

func fromV13Claim(v13 power13.Claim) Claim {
	return Claim{
		RawBytePower:    v13.RawBytePower,
		QualityAdjPower: v13.QualityAdjPower,
	}
}

func (s *state13) ActorKey() string {
	return manifest.PowerKey
}

func (s *state13) ActorVersion() actorstypes.Version {
	return actorstypes.Version13
}

func (s *state13) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

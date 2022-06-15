// Code generated by: `make actors-gen`. DO NOT EDIT.
package power

import (
	"bytes"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin"

	builtin8 "github.com/filecoin-project/specs-actors/v8/actors/builtin"
	power8 "github.com/filecoin-project/specs-actors/v8/actors/builtin/power"
	adt8 "github.com/filecoin-project/specs-actors/v8/actors/util/adt"
)

var _ State = (*state8)(nil)

func load8(store adt.Store, root cid.Cid) (State, error) {
	out := state8{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state8 struct {
	power8.State
	store adt.Store
}

func (s *state8) Code() cid.Cid {
	return builtin8.StoragePowerActorCodeID
}

func (s *state8) TotalLocked() (abi.TokenAmount, error) {
	return s.TotalPledgeCollateral, nil
}

func (s *state8) TotalPower() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalRawBytePower,
		QualityAdjPower: s.TotalQualityAdjPower,
	}, nil
}

// Committed power to the network. Includes miners below the minimum threshold.
func (s *state8) TotalCommitted() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalBytesCommitted,
		QualityAdjPower: s.TotalQABytesCommitted,
	}, nil
}

func (s *state8) MinerPower(addr address.Address) (Claim, bool, error) {
	claims, err := s.claims()
	if err != nil {
		return Claim{}, false, err
	}
	var claim power8.Claim
	ok, err := claims.Get(abi.AddrKey(addr), &claim)
	if err != nil {
		return Claim{}, false, err
	}
	return Claim{
		RawBytePower:    claim.RawBytePower,
		QualityAdjPower: claim.QualityAdjPower,
	}, ok, nil
}

func (s *state8) MinerNominalPowerMeetsConsensusMinimum(a address.Address) (bool, error) {
	return s.State.MinerNominalPowerMeetsConsensusMinimum(s.store, a)
}

func (s *state8) TotalPowerSmoothed() (builtin.FilterEstimate, error) {
	return builtin.FromV8FilterEstimate(s.State.ThisEpochQAPowerSmoothed), nil
}

func (s *state8) MinerCounts() (uint64, uint64, error) {
	return uint64(s.State.MinerAboveMinPowerCount), uint64(s.State.MinerCount), nil
}

func (s *state8) ListAllMiners() ([]address.Address, error) {
	claims, err := s.claims()
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

func (s *state8) ForEachClaim(cb func(miner address.Address, claim Claim) error) error {
	claims, err := s.claims()
	if err != nil {
		return err
	}

	var claim power8.Claim
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

func (s *state8) ClaimsChanged(other State) (bool, error) {
	other8, ok := other.(*state8)
	if !ok {
		// treat an upgrade as a change, always
		return true, nil
	}
	return !s.State.Claims.Equals(other8.State.Claims), nil
}

func (s *state8) claims() (adt.Map, error) {
	return adt8.AsMap(s.store, s.Claims, builtin8.DefaultHamtBitwidth)
}

func (s *state8) decodeClaim(val *cbg.Deferred) (Claim, error) {
	var ci power8.Claim
	if err := ci.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return Claim{}, err
	}
	return fromV8Claim(ci), nil
}

func fromV8Claim(v8 power8.Claim) Claim {
	return Claim{
		RawBytePower:    v8.RawBytePower,
		QualityAdjPower: v8.QualityAdjPower,
	}
}

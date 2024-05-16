// Code generated by: `make actors-gen`. DO NOT EDIT.

package market

import (
	"bytes"
	"fmt"

	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/manifest"
	market6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/market"
	adt6 "github.com/filecoin-project/specs-actors/v6/actors/util/adt"

	lotusactors "github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
)

var _ State = (*state6)(nil)

func load6(store adt.Store, root cid.Cid) (State, error) {
	out := state6{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func make6(store adt.Store) (State, error) {
	out := state6{store: store}

	s, err := market6.ConstructState(store)
	if err != nil {
		return nil, err
	}

	out.State = *s

	return &out, nil
}

type state6 struct {
	market6.State
	store adt.Store
}

func (s *state6) StatesChanged(otherState State) (bool, error) {
	otherState6, ok := otherState.(*state6)
	if !ok {
		// there's no way to compare different versions of the state, so let's
		// just say that means the state of balances has changed
		return true, nil
	}
	return !s.State.States.Equals(otherState6.State.States), nil
}

func (s *state6) States() (DealStates, error) {
	stateArray, err := adt6.AsArray(s.store, s.State.States, market6.StatesAmtBitwidth)
	if err != nil {
		return nil, err
	}
	return &dealStates6{stateArray}, nil
}

func (s *state6) ProposalsChanged(otherState State) (bool, error) {
	otherState6, ok := otherState.(*state6)
	if !ok {
		// there's no way to compare different versions of the state, so let's
		// just say that means the state of balances has changed
		return true, nil
	}
	return !s.State.Proposals.Equals(otherState6.State.Proposals), nil
}

func (s *state6) Proposals() (DealProposals, error) {
	proposalArray, err := adt6.AsArray(s.store, s.State.Proposals, market6.ProposalsAmtBitwidth)
	if err != nil {
		return nil, err
	}
	return &dealProposals6{proposalArray}, nil
}

type dealStates6 struct {
	adt.Array
}

func (s *dealStates6) Get(dealID abi.DealID) (DealState, bool, error) {
	var deal6 market6.DealState
	found, err := s.Array.Get(uint64(dealID), &deal6)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	deal := fromV6DealState(deal6)
	return deal, true, nil
}

func (s *dealStates6) ForEach(cb func(dealID abi.DealID, ds DealState) error) error {
	var ds6 market6.DealState
	return s.Array.ForEach(&ds6, func(idx int64) error {
		return cb(abi.DealID(idx), fromV6DealState(ds6))
	})
}

func (s *dealStates6) decode(val *cbg.Deferred) (DealState, error) {
	var ds6 market6.DealState
	if err := ds6.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return nil, err
	}
	ds := fromV6DealState(ds6)
	return ds, nil
}

func (s *dealStates6) array() adt.Array {
	return s.Array
}

func fromV6DealState(v6 market6.DealState) DealState {
	return dealStateV6{v6}
}

type dealStateV6 struct {
	ds6 market6.DealState
}

func (d dealStateV6) SectorStartEpoch() abi.ChainEpoch {
	return d.ds6.SectorStartEpoch
}

func (d dealStateV6) SectorNumber() abi.SectorNumber {

	return 0

}

func (d dealStateV6) LastUpdatedEpoch() abi.ChainEpoch {
	return d.ds6.LastUpdatedEpoch
}

func (d dealStateV6) SlashEpoch() abi.ChainEpoch {
	return d.ds6.SlashEpoch
}

func (d dealStateV6) Equals(other DealState) bool {
	if ov6, ok := other.(dealStateV6); ok {
		return d.ds6 == ov6.ds6
	}

	if d.SectorStartEpoch() != other.SectorStartEpoch() {
		return false
	}
	if d.LastUpdatedEpoch() != other.LastUpdatedEpoch() {
		return false
	}
	if d.SlashEpoch() != other.SlashEpoch() {
		return false
	}

	return true
}

var _ DealState = (*dealStateV6)(nil)

type dealProposals6 struct {
	adt.Array
}

func (s *dealProposals6) Get(dealID abi.DealID) (*DealProposal, bool, error) {
	var proposal6 market6.DealProposal
	found, err := s.Array.Get(uint64(dealID), &proposal6)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}

	proposal, err := fromV6DealProposal(proposal6)
	if err != nil {
		return nil, true, xerrors.Errorf("decoding proposal: %w", err)
	}

	return &proposal, true, nil
}

func (s *dealProposals6) ForEach(cb func(dealID abi.DealID, dp DealProposal) error) error {
	var dp6 market6.DealProposal
	return s.Array.ForEach(&dp6, func(idx int64) error {
		dp, err := fromV6DealProposal(dp6)
		if err != nil {
			return xerrors.Errorf("decoding proposal: %w", err)
		}

		return cb(abi.DealID(idx), dp)
	})
}

func (s *dealProposals6) decode(val *cbg.Deferred) (*DealProposal, error) {
	var dp6 market6.DealProposal
	if err := dp6.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return nil, err
	}

	dp, err := fromV6DealProposal(dp6)
	if err != nil {
		return nil, err
	}

	return &dp, nil
}

func (s *dealProposals6) array() adt.Array {
	return s.Array
}

func fromV6DealProposal(v6 market6.DealProposal) (DealProposal, error) {

	label, err := labelFromGoString(v6.Label)

	if err != nil {
		return DealProposal{}, xerrors.Errorf("error setting deal label: %w", err)
	}

	return DealProposal{
		PieceCID:     v6.PieceCID,
		PieceSize:    v6.PieceSize,
		VerifiedDeal: v6.VerifiedDeal,
		Client:       v6.Client,
		Provider:     v6.Provider,

		Label: label,

		StartEpoch:           v6.StartEpoch,
		EndEpoch:             v6.EndEpoch,
		StoragePricePerEpoch: v6.StoragePricePerEpoch,

		ProviderCollateral: v6.ProviderCollateral,
		ClientCollateral:   v6.ClientCollateral,
	}, nil
}

func (s *state6) DealProposalsAmtBitwidth() int {
	return market6.ProposalsAmtBitwidth
}

func (s *state6) DealStatesAmtBitwidth() int {
	return market6.StatesAmtBitwidth
}

func (s *state6) ActorKey() string {
	return manifest.MarketKey
}

func (s *state6) ActorVersion() actorstypes.Version {
	return actorstypes.Version6
}

func (s *state6) Code() cid.Cid {
	code, ok := lotusactors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

func (s *state6) GetProviderSectors() (map[abi.SectorID][]abi.DealID, error) {

	return nil, nil

}

func (s *state6) GetProviderSectorsByDealID(dealIDMap map[abi.DealID]bool) (map[abi.DealID]abi.SectorID, error) {

	return nil, nil

}

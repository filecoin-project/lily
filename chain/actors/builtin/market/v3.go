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
	market3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"

	lotusactors "github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
)

var _ State = (*state3)(nil)

func load3(store adt.Store, root cid.Cid) (State, error) {
	out := state3{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func make3(store adt.Store) (State, error) {
	out := state3{store: store}

	s, err := market3.ConstructState(store)
	if err != nil {
		return nil, err
	}

	out.State = *s

	return &out, nil
}

type state3 struct {
	market3.State
	store adt.Store
}

func (s *state3) StatesChanged(otherState State) (bool, error) {
	otherState3, ok := otherState.(*state3)
	if !ok {
		// there's no way to compare different versions of the state, so let's
		// just say that means the state of balances has changed
		return true, nil
	}
	return !s.State.States.Equals(otherState3.State.States), nil
}

func (s *state3) States() (DealStates, error) {
	stateArray, err := adt3.AsArray(s.store, s.State.States, market3.StatesAmtBitwidth)
	if err != nil {
		return nil, err
	}
	return &dealStates3{stateArray}, nil
}

func (s *state3) ProposalsChanged(otherState State) (bool, error) {
	otherState3, ok := otherState.(*state3)
	if !ok {
		// there's no way to compare different versions of the state, so let's
		// just say that means the state of balances has changed
		return true, nil
	}
	return !s.State.Proposals.Equals(otherState3.State.Proposals), nil
}

func (s *state3) Proposals() (DealProposals, error) {
	proposalArray, err := adt3.AsArray(s.store, s.State.Proposals, market3.ProposalsAmtBitwidth)
	if err != nil {
		return nil, err
	}
	return &dealProposals3{proposalArray}, nil
}

type dealStates3 struct {
	adt.Array
}

func (s *dealStates3) Get(dealID abi.DealID) (DealState, bool, error) {
	var deal3 market3.DealState
	found, err := s.Array.Get(uint64(dealID), &deal3)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	deal := fromV3DealState(deal3)
	return deal, true, nil
}

func (s *dealStates3) ForEach(cb func(dealID abi.DealID, ds DealState) error) error {
	var ds3 market3.DealState
	return s.Array.ForEach(&ds3, func(idx int64) error {
		return cb(abi.DealID(idx), fromV3DealState(ds3))
	})
}

func (s *dealStates3) decode(val *cbg.Deferred) (DealState, error) {
	var ds3 market3.DealState
	if err := ds3.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return nil, err
	}
	ds := fromV3DealState(ds3)
	return ds, nil
}

func (s *dealStates3) array() adt.Array {
	return s.Array
}

func fromV3DealState(v3 market3.DealState) DealState {
	return dealStateV3{v3}
}

type dealStateV3 struct {
	ds3 market3.DealState
}

func (d dealStateV3) SectorStartEpoch() abi.ChainEpoch {
	return d.ds3.SectorStartEpoch
}

func (d dealStateV3) LastUpdatedEpoch() abi.ChainEpoch {
	return d.ds3.LastUpdatedEpoch
}

func (d dealStateV3) SlashEpoch() abi.ChainEpoch {
	return d.ds3.SlashEpoch
}

func (d dealStateV3) Equals(other DealState) bool {
	if ov3, ok := other.(dealStateV3); ok {
		return d.ds3 == ov3.ds3
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

var _ DealState = (*dealStateV3)(nil)

type dealProposals3 struct {
	adt.Array
}

func (s *dealProposals3) Get(dealID abi.DealID) (*DealProposal, bool, error) {
	var proposal3 market3.DealProposal
	found, err := s.Array.Get(uint64(dealID), &proposal3)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}

	proposal, err := fromV3DealProposal(proposal3)
	if err != nil {
		return nil, true, xerrors.Errorf("decoding proposal: %w", err)
	}

	return &proposal, true, nil
}

func (s *dealProposals3) ForEach(cb func(dealID abi.DealID, dp DealProposal) error) error {
	var dp3 market3.DealProposal
	return s.Array.ForEach(&dp3, func(idx int64) error {
		dp, err := fromV3DealProposal(dp3)
		if err != nil {
			return xerrors.Errorf("decoding proposal: %w", err)
		}

		return cb(abi.DealID(idx), dp)
	})
}

func (s *dealProposals3) decode(val *cbg.Deferred) (*DealProposal, error) {
	var dp3 market3.DealProposal
	if err := dp3.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return nil, err
	}

	dp, err := fromV3DealProposal(dp3)
	if err != nil {
		return nil, err
	}

	return &dp, nil
}

func (s *dealProposals3) array() adt.Array {
	return s.Array
}

func fromV3DealProposal(v3 market3.DealProposal) (DealProposal, error) {

	label, err := labelFromGoString(v3.Label)

	if err != nil {
		return DealProposal{}, xerrors.Errorf("error setting deal label: %w", err)
	}

	return DealProposal{
		PieceCID:     v3.PieceCID,
		PieceSize:    v3.PieceSize,
		VerifiedDeal: v3.VerifiedDeal,
		Client:       v3.Client,
		Provider:     v3.Provider,

		Label: label,

		StartEpoch:           v3.StartEpoch,
		EndEpoch:             v3.EndEpoch,
		StoragePricePerEpoch: v3.StoragePricePerEpoch,

		ProviderCollateral: v3.ProviderCollateral,
		ClientCollateral:   v3.ClientCollateral,
	}, nil
}

func (s *state3) DealProposalsAmtBitwidth() int {
	return market3.ProposalsAmtBitwidth
}

func (s *state3) DealStatesAmtBitwidth() int {
	return market3.StatesAmtBitwidth
}

func (s *state3) ActorKey() string {
	return manifest.MarketKey
}

func (s *state3) ActorVersion() actorstypes.Version {
	return actorstypes.Version3
}

func (s *state3) Code() cid.Cid {
	code, ok := lotusactors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

func (s *state3) GetProviderSectors() (map[abi.SectorID][]abi.DealID, error) {

	return nil, nil

}

func (s *state3) GetProviderSectorsByDealID(dealIDMap map[abi.DealID]bool) (map[abi.DealID]abi.SectorID, error) {

	return nil, nil

}

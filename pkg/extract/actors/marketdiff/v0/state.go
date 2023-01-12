package v0

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiffResult struct {
	DealStateChanges    DealChangeList
	DealProposalChanges ProposalChangeList
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (cbg.CBORMarshaler, error) {
	out := &StateChange{}
	adtStore := store.WrapBlockStore(ctx, bs)

	if deals := sd.DealStateChanges; deals != nil {
		root, err := deals.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.Deals = root
	}

	if proposals := sd.DealProposalChanges; proposals != nil {
		root, err := proposals.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.Proposals = root
	}
	return out, nil
}

func (sd *StateDiffResult) Kind() string {
	return "market"
}

type StateChange struct {
	Deals     cid.Cid `cborgen:"deals"`
	Proposals cid.Cid `cborgen:"proposals"`
}

type StateDiff struct {
	DiffMethods []actors.ActorStateDiff
}

func (s *StateDiff) State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorDiffResult, error) {
	start := time.Now()
	var stateDiff = new(StateDiffResult)
	for _, f := range s.DiffMethods {
		stateChange, err := f.Diff(ctx, api, act)
		if err != nil {
			return nil, err
		}
		if stateChange == nil {
			continue
		}
		switch stateChange.Kind() {
		case KindMarketDeal:
			stateDiff.DealStateChanges = stateChange.(DealChangeList)
		case KindMarketProposal:
			stateDiff.DealProposalChanges = stateChange.(ProposalChangeList)
		default:
			panic(stateChange.Kind())
		}
	}
	log.Infow("Extracted Market State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}

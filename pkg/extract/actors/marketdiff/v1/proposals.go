package v1

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/market")

type ProposalChange struct {
	DealID   uint64            `cborgen:"dealID"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

func (t *ProposalChange) Key() string {
	return abi.UIntKey(t.DealID).Key()
}

type ProposalChangeList []*ProposalChange

const KindMarketProposal = "market_proposal"

func (p ProposalChangeList) Kind() actors.ActorStateKind {
	return KindMarketProposal
}

func (p ProposalChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range p {
		if err := node.Put(l, l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

type Proposals struct{}

func (p Proposals) Type() string {
	return KindMarketProposal
}

func (Proposals) Diff(ctx context.Context, api tasks.DataSource, act *actors.Change) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMarketProposal, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffProposals(ctx, api, act)
}

func DiffProposals(ctx context.Context, api tasks.DataSource, act *actors.Change) (actors.ActorStateChange, error) {
	arrayChanges, err := generic.DiffActorArray(ctx, api, act, MarketStateLoader, MarketProposlasArrayLoader)
	if err != nil {
		return nil, err
	}
	out := make(ProposalChangeList, len(arrayChanges))
	for i, change := range arrayChanges {
		out[i] = &ProposalChange{
			DealID:   change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		}
	}
	return out, nil
}

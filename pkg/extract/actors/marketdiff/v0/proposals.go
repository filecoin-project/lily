package v0

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/market")

type ProposalChange struct {
	DealID   uint64
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   core.ChangeType
}

type ProposalChangeList []*ProposalChange

const KindMarketProposal = "market_proposal"

func (p ProposalChangeList) Kind() actors.ActorStateKind {
	return KindMarketProposal
}

type Proposals struct{}

func (Proposals) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMarketProposal, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffProposals(ctx, api, act)
}

func DiffProposals(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	arrayChanges, err := generic.DiffActorArray(ctx, api, act, nil, nil)
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

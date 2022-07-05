package miner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type LockedFundsExtractor struct{}

func (LockedFundsExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "LockedFundsExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "LockedFundsExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	currLocked, err := ec.CurrState.LockedFunds()
	if err != nil {
		return nil, fmt.Errorf("loading current miner locked funds: %w", err)
	}
	if ec.HasPreviousState() {
		prevLocked, err := ec.PrevState.LockedFunds()
		if err != nil {
			return nil, fmt.Errorf("loading previous miner locked funds: %w", err)
		}

		// if all values are equal there is no change.
		if prevLocked.VestingFunds.Equals(currLocked.VestingFunds) &&
			prevLocked.PreCommitDeposits.Equals(currLocked.PreCommitDeposits) &&
			prevLocked.InitialPledgeRequirement.Equals(currLocked.InitialPledgeRequirement) {
			return nil, nil
		}
	}
	// funds changed

	return &minermodel.MinerLockedFund{
		Height:            int64(ec.CurrTs.Height()),
		MinerID:           a.Address.String(),
		StateRoot:         a.Current.ParentState().String(),
		LockedFunds:       currLocked.VestingFunds.String(),
		InitialPledge:     currLocked.InitialPledgeRequirement.String(),
		PreCommitDeposits: currLocked.PreCommitDeposits.String(),
	}, nil
}

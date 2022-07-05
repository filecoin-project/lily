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

type DeadlineInfoExtractor struct{}

func (DeadlineInfoExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "DeadlineInfoExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "DeadlineInfoExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}
	currDeadlineInfo, err := ec.CurrState.DeadlineInfo(ec.CurrTs.Height())
	if err != nil {
		return nil, err
	}

	if ec.HasPreviousState() {
		prevDeadlineInfo, err := ec.PrevState.DeadlineInfo(ec.CurrTs.Height())
		if err != nil {
			return nil, err
		}
		// TODO implement equality function
		// dereference pointers to check equality
		// if these are different then return a model in the bottom of function
		if prevDeadlineInfo != nil &&
			currDeadlineInfo != nil &&
			*prevDeadlineInfo == *currDeadlineInfo {
			return nil, nil
		}
	}

	// if there is no previous state and the deadlines have changed, return a model
	return &minermodel.MinerCurrentDeadlineInfo{
		Height:        int64(ec.CurrTs.Height()),
		MinerID:       a.Address.String(),
		StateRoot:     a.Current.ParentState().String(),
		DeadlineIndex: currDeadlineInfo.Index,
		PeriodStart:   int64(currDeadlineInfo.PeriodStart),
		Open:          int64(currDeadlineInfo.Open),
		Close:         int64(currDeadlineInfo.Close),
		Challenge:     int64(currDeadlineInfo.Challenge),
		FaultCutoff:   int64(currDeadlineInfo.FaultCutoff),
	}, nil

}

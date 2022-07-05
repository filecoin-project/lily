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

type FeeDebtExtractor struct{}

func (FeeDebtExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "FeeDebtExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "FeeDebtExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}
	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	currDebt, err := ec.CurrState.FeeDebt()
	if err != nil {
		return nil, fmt.Errorf("loading current miner fee debt: %w", err)
	}

	if ec.HasPreviousState() {
		prevDebt, err := ec.PrevState.FeeDebt()
		if err != nil {
			return nil, fmt.Errorf("loading previous miner fee debt: %w", err)
		}
		if prevDebt.Equals(currDebt) {
			return nil, nil
		}
	}
	// debt changed

	return &minermodel.MinerFeeDebt{
		Height:    int64(ec.CurrTs.Height()),
		MinerID:   a.Address.String(),
		StateRoot: a.Current.ParentState().String(),
		FeeDebt:   currDebt.String(),
	}, nil
}

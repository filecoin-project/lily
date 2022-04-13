package power

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/model"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("lily/tasks/power")

var _ actorstate.ActorStateExtractor = (*ChainPowerExtractor)(nil)

type ChainPowerExtractor struct{}

func (ChainPowerExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "ChainPowerExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "ChainPowerExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewPowerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}
	locked, err := ec.CurrState.TotalLocked()
	if err != nil {
		return nil, err
	}
	pow, err := ec.CurrState.TotalPower()
	if err != nil {
		return nil, err
	}
	commit, err := ec.CurrState.TotalCommitted()
	if err != nil {
		return nil, err
	}
	smoothed, err := ec.CurrState.TotalPowerSmoothed()
	if err != nil {
		return nil, err
	}
	participating, total, err := ec.CurrState.MinerCounts()
	if err != nil {
		return nil, err
	}

	return &powermodel.ChainPower{
		Height:                     int64(ec.CurrTs.Height()),
		StateRoot:                  ec.CurrTs.ParentState().String(),
		TotalRawBytesPower:         pow.RawBytePower.String(),
		TotalQABytesPower:          pow.QualityAdjPower.String(),
		TotalRawBytesCommitted:     commit.RawBytePower.String(),
		TotalQABytesCommitted:      commit.QualityAdjPower.String(),
		TotalPledgeCollateral:      locked.String(),
		QASmoothedPositionEstimate: smoothed.PositionEstimate.String(),
		QASmoothedVelocityEstimate: smoothed.VelocityEstimate.String(),
		MinerCount:                 total,
		ParticipatingMinerCount:    participating,
	}, nil
}

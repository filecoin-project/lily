package actorstate

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	powermodel "github.com/filecoin-project/sentinel-visor/model/actors/power"
)

// was services/processor/tasks/power/power.go

// StoragePowerExtractor extracts power actor state
type StoragePowerExtractor struct{}

func init() {
	Register(sa0builtin.StoragePowerActorCodeID, StoragePowerExtractor{})
	Register(sa2builtin.StoragePowerActorCodeID, StoragePowerExtractor{})
}

func (StoragePowerExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "StoragePowerExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	powerActor, err := node.StateGetActor(ctx, power.Address, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading power actor: %w", err)
	}

	pstate, err := power.Load(node.Store(), powerActor)
	if err != nil {
		return nil, xerrors.Errorf("loading power actor state: %w", err)
	}

	locked, err := pstate.TotalLocked()
	if err != nil {
		return nil, err
	}
	pow, err := pstate.TotalPower()
	if err != nil {
		return nil, err
	}
	commit, err := pstate.TotalCommitted()
	if err != nil {
		return nil, err
	}
	smoothed, err := pstate.TotalPowerSmoothed()
	if err != nil {
		return nil, err
	}
	participating, total, err := pstate.MinerCounts()
	if err != nil {
		return nil, err
	}

	return &powermodel.ChainPower{
		StateRoot:                  a.ParentStateRoot.String(),
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

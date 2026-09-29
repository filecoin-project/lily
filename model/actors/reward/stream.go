package reward

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

// ChainRewardStream is one epoch of FIP-0118 block-reward stream state from the reward actor (f02).
// Weights are DENOM (1e18) fixed-point integers. Burn is the residual and has no stream record.
type ChainRewardStream struct {
	tableName struct{} `pg:"chain_reward_streams"` // nolint: structcheck

	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	// TotalMintedReward is all FIL minted through block rewards at this epoch.
	TotalMintedReward string `pg:"type:numeric,notnull"`
	// TotalBurnMinted is the cumulative block-reward residual sent to the burn actor.
	TotalBurnMinted string `pg:"type:numeric,notnull"`
	// TotalExplicitMinted is the cumulative block reward accrued to explicit streams.
	TotalExplicitMinted string `pg:"type:numeric,notnull"`

	// BurnWeight is w0 at this epoch, in DENOM fixed point.
	BurnWeight string `pg:"type:numeric,notnull"`
	// ConsensusWeight is w1 (stream id 1) at this epoch, in DENOM fixed point.
	ConsensusWeight string `pg:"type:numeric,notnull"`
	// ServiceWeight is w2 (stream id 2) at this epoch, in DENOM fixed point.
	ServiceWeight string `pg:"type:numeric,notnull"`

	// ConsensusVStart is the consensus stream weight at ConsensusTStart, in DENOM fixed point.
	ConsensusVStart string `pg:"type:numeric,notnull"`
	// ConsensusSlope is the consensus stream weight change per epoch, in DENOM fixed point.
	ConsensusSlope string `pg:"type:numeric,notnull"`
	// ConsensusTStart is the epoch at which ConsensusVStart applies.
	ConsensusTStart int64 `pg:",notnull,use_zero"`
	// ConsensusFloor is the consensus stream lower clamp, in DENOM fixed point.
	ConsensusFloor string `pg:"type:numeric,notnull"`
	// ConsensusCap is the consensus stream upper clamp, in DENOM fixed point.
	ConsensusCap string `pg:"type:numeric,notnull"`

	// ServiceVStart is the service stream weight at ServiceTStart, in DENOM fixed point.
	ServiceVStart string `pg:"type:numeric,notnull"`
	// ServiceSlope is the service stream weight change per epoch, in DENOM fixed point.
	ServiceSlope string `pg:"type:numeric,notnull"`
	// ServiceTStart is the epoch at which ServiceVStart applies.
	ServiceTStart int64 `pg:",notnull,use_zero"`
	// ServiceFloor is the service stream lower clamp, in DENOM fixed point.
	ServiceFloor string `pg:"type:numeric,notnull"`
	// ServiceCap is the service stream upper clamp, in DENOM fixed point.
	ServiceCap string `pg:"type:numeric,notnull"`
}

func (r *ChainRewardStream) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ChainRewardStream.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_reward_streams"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, r)
}

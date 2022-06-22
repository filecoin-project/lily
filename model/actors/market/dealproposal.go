package market

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

// MarketDealProposalV1_7 is exported from the market actor iff the actor code is less than v8.
// MarketDealProposalV1_7 contains all storage deal states with latest values applied to end_epoch when updates are detected on-chain.
type MarketDealProposalV1_7 struct {
	tableName struct{} `pg:"market_deal_proposals"` // nolint: structcheck
	// Epoch at which this deal proposal was added or changed.
	Height int64 `pg:",pk,notnull,use_zero"`
	// Identifier for the deal.
	DealID uint64 `pg:",pk,use_zero"`
	// CID of the parent state root for this deal.
	StateRoot string `pg:",notnull"`

	// The piece size in bytes with padding.
	PaddedPieceSize uint64 `pg:",use_zero"`
	// The piece size in bytes without padding.
	UnpaddedPieceSize uint64 `pg:",use_zero"`

	// The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid.
	StartEpoch int64 `pg:",use_zero"`
	// The epoch at which this deal with end.
	EndEpoch int64 `pg:",use_zero"`

	// Address of the actor proposing the deal.
	ClientID string `pg:",notnull"`
	// Address of the actor providing the services.
	ProviderID string `pg:",notnull"`
	// The amount of FIL (in attoFIL) the client has pledged as collateral.
	ClientCollateral string `pg:",notnull"`
	// The amount of FIL (in attoFIL) the provider has pledged as collateral. The Provider deal collateral is only slashed when a sector is terminated before the deal expires.
	ProviderCollateral string `pg:",notnull"`
	// The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for.
	StoragePricePerEpoch string `pg:",notnull"`
	// CID of a sector piece. A Piece is an object that represents a whole or part of a File.
	PieceCID string `pg:",notnull"`

	// Deal is with a verified provider.
	IsVerified bool `pg:",notnull,use_zero"`
	// An arbitrary client chosen label to apply to the deal.
	Label string
}

func (dp *MarketDealProposalV1_7) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_proposals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, dp)
}

type MarketDealProposalsV1_7 []*MarketDealProposalV1_7

func (dps MarketDealProposalsV1_7) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MarketDealProposalsV1_7.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(dps)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_proposals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(dps))
	return s.PersistModel(ctx, dps)
}

// MarketDealProposalV8 is the default model exported fromt he market actor.
// The table is returned iff the market actor code is greater than or equal to v8.
// The table contains all storage deal states with the latest values applied to end_epoch when updates are detected on-chain.
type MarketDealProposalV8 struct {
	tableName struct{} `pg:"market_deal_proposals_v8"`
	// Epoch at which this deal proposal was added or changed.
	Height int64 `pg:",pk,notnull,use_zero"`
	// Identifier for the deal.
	DealID uint64 `pg:",pk,use_zero"`
	// CID of the parent state root for this deal.
	StateRoot string `pg:",notnull"`

	// The piece size in bytes with padding.
	PaddedPieceSize uint64 `pg:",use_zero"`
	// The piece size in bytes without padding.
	UnpaddedPieceSize uint64 `pg:",use_zero"`

	// The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid.
	StartEpoch int64 `pg:",use_zero"`
	// The epoch at which this deal with end.
	EndEpoch int64 `pg:",use_zero"`

	// Address of the actor proposing the deal.
	ClientID string `pg:",notnull"`
	// Address of the actor providing the services.
	ProviderID string `pg:",notnull"`
	// The amount of FIL (in attoFIL) the client has pledged as collateral.
	ClientCollateral string `pg:",notnull"`
	// The amount of FIL (in attoFIL) the provider has pledged as collateral. The Provider deal collateral is only slashed when a sector is terminated before the deal expires.
	ProviderCollateral string `pg:",notnull"`
	// The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for.
	StoragePricePerEpoch string `pg:",notnull"`
	// CID of a sector piece. A Piece is an object that represents a whole or part of a File.
	PieceCID string `pg:",notnull"`

	// Deal is with a verified provider.
	IsVerified bool `pg:",notnull,use_zero"`

	// A base64 encoded string chosen by the client.
	// Related to FIP: https://github.com/filecoin-project/FIPs/blob/master/FIPS/fip-0027.md
	Label string

	// When true Label contains a valid UTF-8 string encoded in base64. When false Label contains raw bytes encoded in base64.
	// Related to FIP: https://github.com/filecoin-project/FIPs/blob/master/FIPS/fip-0027.md
	IsString bool `pg:",notnull"`
}

func (dp *MarketDealProposalV8) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_proposals_v8"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, dp)
}

type MarketDealProposalsV8 []*MarketDealProposalV8

func (dps MarketDealProposalsV8) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MarketDealProposalsV8.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(dps)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_proposals_v8"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(dps))
	return s.PersistModel(ctx, dps)
}

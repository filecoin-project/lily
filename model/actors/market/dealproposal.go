package market

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

// MarketDealProposal contains all storage deal states with latest values applied to end_epoch when updates are detected on-chain.
type MarketDealProposal struct {
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
	// An arbitrary client chosen label to apply to the deal. The value is base64 encoded before persisting.
	Label string

	// When true Label contains a valid UTF-8 string encoded in base64. When false Label contains raw bytes encoded in base64. Related to FIP: https://github.com/filecoin-project/FIPs/blob/master/FIPS/fip-0027.md
	IsString bool
}

func (dp *MarketDealProposal) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_proposals"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, dp)
}

type MarketDealProposals []*MarketDealProposal

func (dps MarketDealProposals) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MarketDealProposals.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(dps)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_proposals"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(dps))
	return s.PersistModel(ctx, dps)
}

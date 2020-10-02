package market

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

type MarketDealProposal struct {
	DealID    uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",notnull"`

	PaddedPieceSize   uint64 `pg:",use_zero"`
	UnpaddedPieceSize uint64 `pg:",use_zero"`

	StartEpoch int64 `pg:",use_zero"`
	EndEpoch   int64 `pg:",use_zero"`

	ClientID             string `pg:",notnull"`
	ProviderID           string `pg:",notnull"`
	ClientCollateral     string `pg:",notnull"`
	ProviderCollateral   string `pg:",notnull"`
	StoragePricePerEpoch string `pg:",notnull"`
	PieceCID             string `pg:",notnull"`

	IsVerified bool `pg:",notnull"`
	Label      string
}

func (dp *MarketDealProposal) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, dp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

type MarketDealProposals []*MarketDealProposal

func (dps MarketDealProposals) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MarketDealProposals.PersistWithTx", trace.WithAttributes(label.Int("count", len(dps))))
	defer span.End()
	for _, dp := range dps {
		if err := dp.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

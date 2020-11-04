package power

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

type PowerActorClaim struct {
	Height          int64  `pg:",pk,notnull,use_zero"`
	MinerID         string `pg:",pk,notnull"`
	StateRoot       string `pg:",pk,notnull"`
	RawBytePower    string `pg:",notnull"`
	QualityAdjPower string `pg:",notnull"`
}

func (p *PowerActorClaim) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaim.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, p).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting power actors claim: %w", err)
	}
	return nil
}

type PowerActorClaimList []*PowerActorClaim

func (pl PowerActorClaimList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaimList.PersistWithTx")
	defer span.End()
	if len(pl) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &pl).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting power actor claim list: %w")
	}
	return nil

}

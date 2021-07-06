package extractors

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/power"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/power/extract"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

func init() {
	extract.Register(&PowerActorClaim{}, ExtractClaimedPower)
}

func ExtractClaimedPower(ctx context.Context, ec *extract.PowerStateExtractionContext) (model.Persistable, error) {
	claimModel := PowerActorClaimList{}
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachClaim(func(miner address.Address, claim power.Claim) error {
			claimModel = append(claimModel, &PowerActorClaim{
				Height:          int64(ec.CurrTs.Height()),
				StateRoot:       ec.CurrTs.ParentState().String(),
				MinerID:         miner.String(),
				RawBytePower:    claim.RawBytePower.String(),
				QualityAdjPower: claim.QualityAdjPower.String(),
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return claimModel, nil
	}
	// normal case.
	claimChanges, err := power.DiffClaims(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	for _, newClaim := range claimChanges.Added {
		claimModel = append(claimModel, &PowerActorClaim{
			Height:          int64(ec.CurrTs.Height()),
			StateRoot:       ec.CurrTs.ParentState().String(),
			MinerID:         newClaim.Miner.String(),
			RawBytePower:    newClaim.Claim.RawBytePower.String(),
			QualityAdjPower: newClaim.Claim.QualityAdjPower.String(),
		})
	}
	for _, modClaim := range claimChanges.Modified {
		claimModel = append(claimModel, &PowerActorClaim{
			Height:          int64(ec.CurrTs.Height()),
			StateRoot:       ec.CurrTs.ParentState().String(),
			MinerID:         modClaim.Miner.String(),
			RawBytePower:    modClaim.To.RawBytePower.String(),
			QualityAdjPower: modClaim.To.QualityAdjPower.String(),
		})
	}
	return claimModel, nil
}

type PowerActorClaim struct {
	Height          int64  `pg:",pk,notnull,use_zero"`
	MinerID         string `pg:",pk,notnull"`
	StateRoot       string `pg:",pk,notnull"`
	RawBytePower    string `pg:"type:numeric,notnull"`
	QualityAdjPower string `pg:"type:numeric,notnull"`
}

type PowerActorClaimV0 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName       struct{} `pg:"power_actor_claims"`
	Height          int64    `pg:",pk,notnull,use_zero"`
	MinerID         string   `pg:",pk,notnull"`
	StateRoot       string   `pg:",pk,notnull"`
	RawBytePower    string   `pg:",notnull"`
	QualityAdjPower string   `pg:",notnull"`
}

func (p *PowerActorClaim) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if p == nil {
			return (*PowerActorClaimV0)(nil), true
		}

		return &PowerActorClaimV0{
			Height:          p.Height,
			MinerID:         p.MinerID,
			StateRoot:       p.StateRoot,
			RawBytePower:    p.RawBytePower,
			QualityAdjPower: p.QualityAdjPower,
		}, true
	case 1:
		return p, true
	default:
		return nil, false
	}
}

func (p *PowerActorClaim) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaim.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "power_actor_claims"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	vp, ok := p.AsVersion(version)
	if !ok {
		return xerrors.Errorf("PowerActorClaim not supported for schema version %s", version)
	}

	return s.PersistModel(ctx, vp)
}

type PowerActorClaimList []*PowerActorClaim

func (pl PowerActorClaimList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaimList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "power_actor_claims"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(pl) == 0 {
		return nil
	}

	if version.Major != 1 {
		// Support older versions, but in a non-optimal way
		for _, m := range pl {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	return s.PersistModel(ctx, pl)
}

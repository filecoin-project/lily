package chain

import (
	"context"
	"fmt"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type ChainEconomics struct {
	tableName           struct{} `pg:"chain_economics"` // nolint: structcheck
	Height              int64    `pg:",pk,notnull,use_zero"`
	ParentStateRoot     string   `pg:",pk,notnull"`
	CirculatingFil      string   `pg:"type:numeric,notnull"`
	VestedFil           string   `pg:"type:numeric,notnull"`
	MinedFil            string   `pg:"type:numeric,notnull"`
	BurntFil            string   `pg:"type:numeric,notnull"`
	LockedFil           string   `pg:"type:numeric,notnull"`
	FilReserveDisbursed string   `pg:"type:numeric,notnull"`
	LockedFilV2         string   `pg:"type:numeric,notnull"`
}

type ChainEconomicsV0 struct {
	tableName       struct{} `pg:"chain_economics"` // nolint: structcheck
	ParentStateRoot string   `pg:",notnull"`
	CirculatingFil  string   `pg:",notnull"`
	VestedFil       string   `pg:",notnull"`
	MinedFil        string   `pg:",notnull"`
	BurntFil        string   `pg:",notnull"`
	LockedFil       string   `pg:",notnull"`
}

func (c *ChainEconomics) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if c == nil {
			return (*ChainEconomicsV0)(nil), true
		}

		return &ChainEconomicsV0{
			ParentStateRoot: c.ParentStateRoot,
			CirculatingFil:  c.CirculatingFil,
			VestedFil:       c.VestedFil,
			MinedFil:        c.MinedFil,
			BurntFil:        c.BurntFil,
			LockedFil:       c.LockedFil,
		}, true
	case 1:
		return c, true
	default:
		return nil, false
	}
}

func (c *ChainEconomics) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_economics"))

	m, ok := c.AsVersion(version)
	if !ok {
		return fmt.Errorf("ChainEconomics not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type ChainEconomicsList []*ChainEconomics

func (l ChainEconomicsList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "ChainEconomicsList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(l)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_economics"))

	if version.Major != 1 {
		// Support older versions, but in a non-optimal way
		for _, m := range l {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	metrics.RecordCount(ctx, metrics.PersistModel, len(l))
	return s.PersistModel(ctx, l)
}

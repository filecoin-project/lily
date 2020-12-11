package blocks

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

func NewDrandBlockEntries(header *types.BlockHeader) DrandBlockEntries {
	var out DrandBlockEntries
	for _, ent := range header.BeaconEntries {
		out = append(out, &DrandBlockEntrie{
			Round: ent.Round,
			Block: header.Cid().String(),
		})
	}
	return out
}

type DrandBlockEntrie struct {
	Round uint64 `pg:",pk,use_zero"`
	Block string `pg:",notnull"`
}

func (dbe *DrandBlockEntrie) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, dbe)
}

type DrandBlockEntries []*DrandBlockEntrie

func (dbes DrandBlockEntries) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(dbes) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "DrandBlockEntries.Persist", trace.WithAttributes(label.Int("count", len(dbes))))
	defer span.End()
	return s.PersistModel(ctx, dbes)
}

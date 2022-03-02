package datasource

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/metrics"
)

func (t *DataSource) DiffSectors(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.SectorChanges, error) {
	metrics.RecordInc(ctx, metrics.DataSourceSectorDiffCacheHit)
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.DiffSectors")
	defer span.End()

	curA, err := t.Actor(ctx, addr, ts.Key())
	if err != nil {
		return nil, err
	}
	preA, err := t.Actor(ctx, addr, pts.Key())
	if err != nil {
		return nil, err
	}
	key := curA.Head.String() + preA.Head.String()
	value, found := t.diffSectorsCache.Get(key)
	if found {
		metrics.RecordInc(ctx, metrics.DataSourceSectorDiffCacheHit)
		return value.(*miner.SectorChanges), nil
	}

	value, err, shared := t.diffSectorsGroup.Do(key, func() (interface{}, error) {
		data, innerErr := miner.DiffSectors(ctx, t.Store(), pre, cur)
		if innerErr == nil {
			t.diffSectorsCache.Add(key, data)
		}

		return data, innerErr
	})
	if span.IsRecording() {
		span.SetAttributes(attribute.Bool("shared", shared))
	}
	if err != nil {
		return nil, err
	}
	return value.(*miner.SectorChanges), nil
}

func (t *DataSource) DiffPreCommits(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChanges, error) {
	metrics.RecordInc(ctx, metrics.DataSourcePreCommitDiffRead)
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.DiffPreCommits")
	defer span.End()

	curA, err := t.Actor(ctx, addr, ts.Key())
	if err != nil {
		return nil, err
	}
	preA, err := t.Actor(ctx, addr, pts.Key())
	if err != nil {
		return nil, err
	}
	key := curA.Head.String() + preA.Head.String()
	value, found := t.diffPreCommitCache.Get(key)
	if found {
		metrics.RecordInc(ctx, metrics.DataSourcePreCommitDiffCacheHit)
		return value.(*miner.PreCommitChanges), nil
	}

	value, err, shared := t.diffPreCommitGroup.Do(key, func() (interface{}, error) {
		data, innerErr := miner.DiffPreCommits(ctx, t.Store(), pre, cur)
		if innerErr == nil {
			t.diffPreCommitCache.Add(key, data)
		}

		return data, innerErr
	})
	if span.IsRecording() {
		span.SetAttributes(attribute.Bool("shared", shared))
	}
	if err != nil {
		return nil, err
	}
	return value.(*miner.PreCommitChanges), nil

}

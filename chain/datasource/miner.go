package datasource

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/go-address"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/metrics"

	"github.com/filecoin-project/lotus/chain/types"
)

func (t *DataSource) DiffSectors(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.SectorChanges, error) {
	metrics.RecordInc(ctx, metrics.DataSourceSectorDiffRead)
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.DiffSectors")
	defer span.End()

	key, err := asKey(addr, ts, pts)
	if err != nil {
		return nil, err
	}
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

	key, err := asKey(addr, ts, pts)
	if err != nil {
		return nil, err
	}
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

func (t *DataSource) DiffPreCommitsV8(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChangesV8, error) {
	if pre.ActorVersion() > actorstypes.Version8 {
		return nil, fmt.Errorf("cannot diff pre actor version %d using DiffPreCommitsV8 method", pre.ActorVersion())
	}
	if cur.ActorVersion() > actorstypes.Version8 {
		return nil, fmt.Errorf("cannot diff cur actor version %d using DiffPreCommitsV8 method", cur.ActorVersion())
	}
	metrics.RecordInc(ctx, metrics.DataSourcePreCommitDiffRead)
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.DiffPreCommitsV8")
	defer span.End()

	key, err := asKey(addr, ts, pts)
	if err != nil {
		return nil, err
	}
	value, found := t.diffPreCommitCache.Get(key)
	if found {
		metrics.RecordInc(ctx, metrics.DataSourcePreCommitDiffCacheHit)
		return value.(*miner.PreCommitChangesV8), nil
	}

	value, err, shared := t.diffPreCommitGroup.Do(key, func() (interface{}, error) {
		data, innerErr := miner.DiffPreCommitsV8(ctx, t.Store(), pre, cur)
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
	return value.(*miner.PreCommitChangesV8), nil

}

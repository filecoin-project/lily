package diff

import (
	"context"
	"os"
	"runtime"
	"strconv"

	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/go-hamt-ipld/v3"
	adt2 "github.com/filecoin-project/lily/chain/actors/adt"

	"github.com/filecoin-project/lotus/chain/actors/adt"
)

var HamtParallelWorkerLimit int64

const HamtParallelWorkerEnv = "LILY_HAMT_PARALLEL_WORKER_LIMIT"

func init() {
	HamtParallelWorkerLimit = int64(runtime.NumCPU())
	workerStr := os.Getenv(HamtParallelWorkerEnv)
	if workerStr != "" {
		HamtParallelWorkerLimit, err := strconv.ParseInt(workerStr, 10, 64)
		if err != nil {
			log.Warnf("failed to parse env %s defaulting to %d (runtime.NumCPU()) : %s", HamtParallelWorkerEnv, HamtParallelWorkerLimit, err)
		}
	}
}

// Hamt returns a set of changes that transform `preMap` into `curMap`. opts are applied to both `preMap` and `curMap`.
func Hamt(ctx context.Context, preMap, curMap adt2.Map, preStore, curStore adt.Store, hamtOpts ...hamt.Option) ([]*hamt.Change, error) {
	ctx, span := otel.Tracer("").Start(ctx, "Hamt.Diff")
	defer span.End()

	preRoot, err := preMap.Root()
	if err != nil {
		return nil, err
	}

	curRoot, err := curMap.Root()
	if err != nil {
		return nil, err
	}

	return hamt.ParallelDiff(ctx, preStore, curStore, preRoot, curRoot, HamtParallelWorkerLimit, hamtOpts...)
}

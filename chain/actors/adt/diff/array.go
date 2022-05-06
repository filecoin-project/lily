package diff

import (
	"context"
	"os"
	"runtime"
	"strconv"

	"github.com/filecoin-project/go-amt-ipld/v4"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"go.opentelemetry.io/otel"

	adt2 "github.com/filecoin-project/lily/chain/actors/adt"
)

var AmtParallelWorkerLimit int64

const AmtParallelWorkerEnv = "LILY_AMT_PARALLEL_WORKER_LIMIT"

func init() {
	AmtParallelWorkerLimit = int64(runtime.NumCPU())
	workerStr := os.Getenv(AmtParallelWorkerEnv)
	if workerStr != "" {
		AmtParallelWorkerLimit, err := strconv.ParseInt(workerStr, 10, 64)
		if err != nil {
			log.Warnf("failed to parse env %s defaulting to %d (runtime.NumCPU()) : %s", AmtParallelWorkerEnv, AmtParallelWorkerLimit, err)
		}
	}
}

// Amt returns a set of changes that transform `preArr` into `curArr`. opts are applied to both `preArr` and `curArr`.
func Amt(ctx context.Context, preArr, curArr adt2.Array, preStore, curStore adt.Store, amtOpts ...amt.Option) ([]*amt.Change, error) {
	ctx, span := otel.Tracer("").Start(ctx, "Amt.Diff")
	defer span.End()

	preRoot, err := preArr.Root()
	if err != nil {
		return nil, err
	}

	curRoot, err := curArr.Root()
	if err != nil {
		return nil, err
	}

	return amt.ParallelDiff(ctx, preStore, curStore, preRoot, curRoot, AmtParallelWorkerLimit, amtOpts...)
}

//go:build integration
// +build integration

package itests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	tstorage "github.com/filecoin-project/lily/storage/testing"
)

func TestCalibrationVector(t *testing.T) {
	logging.SetAllLoggers(logging.LevelInfo)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	strg, strgCleanup := tstorage.WaitForExclusiveMigratedStorage(ctx, t, false)
	t.Cleanup(func() {
		err := strgCleanup()
		require.NoError(t, err)
	})

	for _, vf := range CalibnetTestVectors {
		t.Run(filepath.Base(vf.File.Name()), func(t *testing.T) {
			tvb := NewVectorWalkValidatorBuilder(vf).
				WithDatabase(strg).
				WithRange(vf.From, vf.To).
				WithTasks(tasktype.ActorStatesRawTask, tasktype.BlocksTask, tasktype.MessagesTask, tasktype.ChainConsensusTask)

			vw := tvb.Build(ctx, t)
			stop := vw.Run(ctx)
			vw.Validate(t)
			require.NoError(t, stop(ctx))
			require.NoError(t, vf.Close())
		})
	}
}

package rewarddistribution

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-state-types/big"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/testutil"

	"github.com/filecoin-project/lotus/api/v2api"
	"github.com/filecoin-project/lotus/chain/types"
)

type fakeDataSource struct {
	tasks.DataSource
	distribution *v2api.RewardDistribution
	err          error
	executed     *types.TipSet
}

func (f *fakeDataSource) RewardDistribution(_ context.Context, ts *types.TipSet) (*v2api.RewardDistribution, error) {
	f.executed = ts
	return f.distribution, f.err
}

func TestProcessTipSets(t *testing.T) {
	current := testutil.MustFakeTipSet(t, 11)
	executed := testutil.MustFakeTipSet(t, 10)

	t.Run("stores the distribution of the executed tipset", func(t *testing.T) {
		node := &fakeDataSource{distribution: &v2api.RewardDistribution{
			TipSetKey: executed.Key(),
			Height:    executed.Height(),
			Denom:     1_000_000_000_000_000_000,
			Totals:    v2api.RewardAmounts{MintedReward: big.MustFromString("340282366920938463463374607431768211456")},
		}}

		data, report, err := NewTask(node).ProcessTipSets(context.Background(), current, executed)
		require.NoError(t, err)
		require.Same(t, executed, node.executed)
		require.Equal(t, int64(current.Height()), report.Height)
		require.Nil(t, report.ErrorsDetected)

		row, ok := data.(*rewardmodel.ChainRewardDistribution)
		require.True(t, ok)
		require.Equal(t, int64(executed.Height()), row.Height)
		require.Equal(t, executed.Key().String(), row.TipSetKey)

		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(row.Distribution), &decoded))
		require.Equal(t, "1000000000000000000", decoded["Denom"])
		require.Equal(t, "340282366920938463463374607431768211456", decoded["Totals"].(map[string]interface{})["MintedReward"])
	})

	t.Run("reports unsupported network versions without data", func(t *testing.T) {
		data, report, err := NewTask(&fakeDataSource{}).ProcessTipSets(context.Background(), current, executed)
		require.NoError(t, err)
		require.Nil(t, data)
		require.Equal(t, unsupportedInformation, report.StatusInformation)
		require.Nil(t, report.ErrorsDetected)
	})

	t.Run("reports lookup errors", func(t *testing.T) {
		data, report, err := NewTask(&fakeDataSource{err: errors.New("boom")}).ProcessTipSets(context.Background(), current, executed)
		require.NoError(t, err)
		require.Nil(t, data)
		require.Error(t, report.ErrorsDetected.(error))
	})
}

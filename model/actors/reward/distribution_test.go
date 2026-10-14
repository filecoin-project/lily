package reward

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/model"
)

type distributionStorage struct {
	value interface{}
}

func (s *distributionStorage) PersistModel(_ context.Context, value interface{}) error {
	s.value = value
	return nil
}

func TestChainRewardDistributionPersist(t *testing.T) {
	row := &ChainRewardDistribution{
		Height:       42,
		TipSetKey:    "tipset",
		Distribution: `{"Blocks":[{"Streams":[{"Weight":"1000000000000000000"}]}]}`,
	}
	storage := &distributionStorage{}
	require.NoError(t, row.Persist(context.Background(), storage, model.Version{Major: 1, Patch: 48}))
	require.Same(t, row, storage.value)
}

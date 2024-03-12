package testing

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/integrated/tipset"
	"github.com/filecoin-project/lily/model"

	"github.com/filecoin-project/lotus/chain/types"
)

//revive:disable
type MockIndexBuilder struct {
	MockIndexer *MockIndexer
	mock.Mock
}

func (t *MockIndexBuilder) Name() string {
	return "mockindexbuilder"
}

func (t *MockIndexBuilder) WithTasks(_ []string) tipset.IndexerBuilder {
	return t
}

func (t *MockIndexBuilder) WithInterval(_ int) tipset.IndexerBuilder {
	return t
}

func (t *MockIndexBuilder) Build() (tipset.Indexer, error) {
	return t.MockIndexer, nil
}

type MockIndexer struct {
	mock.Mock
}

func (t *MockIndexer) TipSet(ctx context.Context, ts *types.TipSet) (chan *tipset.Result, chan error, error) {
	args := t.Called(ctx, ts)
	resChan := args.Get(0)
	errChan := args.Get(1)
	err := args.Error(2)
	return resChan.(chan *tipset.Result), errChan.(chan error), err
}

type MockExporter struct {
	mock.Mock
}

func (t *MockExporter) ExportResult(ctx context.Context, strg model.Storage, height int64, m []*indexer.ModelResult) error {
	args := t.Called(ctx, strg, height, m)
	return args.Error(0)
}

type FakeStorage struct {
}

func (t *FakeStorage) PersistBatch(ctx context.Context, ps ...model.Persistable) error {
	return nil
}

type FakePersistable struct {
}

func (t *FakePersistable) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	return nil
}

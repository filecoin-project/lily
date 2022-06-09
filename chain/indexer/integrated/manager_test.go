package integrated

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/indexer"
	test "github.com/filecoin-project/lily/chain/indexer/integrated/testing"
	"github.com/filecoin-project/lily/chain/indexer/integrated/tipset"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

func TestManagerStatusOK(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 1)
	errChan := make(chan error)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// create some fake data and a processing report
	data := &test.FakePersistable{}
	report := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "task",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status of OK means indexing was successful
			Status:            visormodel.ProcessingStatusOK,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send report to index manager and close the channel to signal MockIndexer is done indexing data.
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: report,
	}
	close(resChan)
	close(errChan)

	// mock exporter expects to receieve a result with data and report
	mExporter.On("ExportResult", mock.Anything, fStorage, int64(ts1.Height()), []*indexer.ModelResult{
		{
			Name:  t.Name(),
			Model: model.PersistableList{report, data},
		},
	}).Return(nil)

	success, err := manager.TipSet(ctx, ts1)
	require.NoError(t, err)
	require.True(t, success)
}

func TestManagerStatusInfo(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 1)
	errChan := make(chan error)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// create some fake data and a processing report
	data := &test.FakePersistable{}
	report := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "task",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status of Info means indexing was successful
			Status:            visormodel.ProcessingStatusInfo,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send report to index manager and close the channel to signal MockIndexer is done indexing data.
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: report,
	}
	close(resChan)
	close(errChan)

	// mock exporter expects to receieve a result with data and report
	mExporter.On("ExportResult", mock.Anything, fStorage, int64(ts1.Height()), []*indexer.ModelResult{
		{
			Name:  t.Name(),
			Model: model.PersistableList{report, data},
		},
	}).Return(nil)

	success, err := manager.TipSet(ctx, ts1)
	require.NoError(t, err)
	require.True(t, success)
}

func TestManagerStatusOKAndError(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 2)
	errChan := make(chan error)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// create some fake data and a processing report
	data := &test.FakePersistable{}
	reportOK := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "taskOk",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status of OK means indexing was successful
			Status:            visormodel.ProcessingStatusOK,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send report to index manager
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: reportOK,
	}
	reportError := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "taskError",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status of Error means indexing was unsuccessful
			Status:            visormodel.ProcessingStatusError,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send to index manager
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: reportError,
	}
	close(resChan)
	close(errChan)

	// mock exporter expects to receieve two results
	mExporter.On("ExportResult", mock.Anything, fStorage, int64(ts1.Height()), []*indexer.ModelResult{
		{
			Name:  t.Name(),
			Model: model.PersistableList{reportOK, data},
		},
		{
			Name:  t.Name(),
			Model: model.PersistableList{reportError, data},
		},
	}).Return(nil)

	success, err := manager.TipSet(ctx, ts1)
	require.NoError(t, err)
	require.False(t, success)

}

func TestManagerStatusOKAndSkip(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 2)
	errChan := make(chan error)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// create some fake data and a processing report
	data := &test.FakePersistable{}
	reportOK := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "taskOk",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status of OK means indexing was successful
			Status:            visormodel.ProcessingStatusOK,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send report to index manager
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: reportOK,
	}
	reportError := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "taskError",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status of Skip means indexing was unsuccessful
			Status:            visormodel.ProcessingStatusSkip,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send to index manager
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: reportError,
	}
	close(resChan)
	close(errChan)

	// mock exporter expects to receieve two results
	mExporter.On("ExportResult", mock.Anything, fStorage, int64(ts1.Height()), []*indexer.ModelResult{
		{
			Name:  t.Name(),
			Model: model.PersistableList{reportOK, data},
		},
		{
			Name:  t.Name(),
			Model: model.PersistableList{reportError, data},
		},
	}).Return(nil)

	success, err := manager.TipSet(ctx, ts1)
	require.NoError(t, err)
	require.False(t, success)

}

func TestManagerStatusError(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 1)
	errChan := make(chan error)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// create some fake data and a processing report
	data := &test.FakePersistable{}
	report := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "task",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status Error means indexing was unsuccessful
			Status:            visormodel.ProcessingStatusError,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send report to index manager and close the channel to signal MockIndexer is done indexing data.
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: report,
	}
	close(resChan)
	close(errChan)

	// mock exporter expects to receieve a result with data and report
	mExporter.On("ExportResult", mock.Anything, fStorage, int64(ts1.Height()), []*indexer.ModelResult{
		{
			Name:  t.Name(),
			Model: model.PersistableList{report, data},
		},
	}).Return(nil)

	success, err := manager.TipSet(ctx, ts1)
	require.NoError(t, err)
	require.False(t, success)
}

func TestManagerStatusSkip(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 1)
	errChan := make(chan error)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// create some fake data and a processing report
	data := &test.FakePersistable{}
	report := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "task",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status Skip means indexing was unsuccessful
			Status:            visormodel.ProcessingStatusSkip,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send report to index manager and close the channel to signal MockIndexer is done indexing data.
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: report,
	}
	close(resChan)
	close(errChan)

	// mock exporter expects to receieve a result with data and report
	mExporter.On("ExportResult", mock.Anything, fStorage, int64(ts1.Height()), []*indexer.ModelResult{
		{
			Name:  t.Name(),
			Model: model.PersistableList{report, data},
		},
	}).Return(nil)

	success, err := manager.TipSet(ctx, ts1)
	require.NoError(t, err)
	require.False(t, success)
}

func TestManagerFatalError(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 1)
	errChan := make(chan error, 1)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// send a fatal error to the index manager
	errChan <- fmt.Errorf("fatal error")
	close(resChan)
	close(errChan)

	success, err := manager.TipSet(ctx, ts1)
	require.Error(t, err)
	require.False(t, success)
}

func TestManagerFatalErrorAndOkReport(t *testing.T) {
	ctx := context.Background()

	// mock index builder and mock indexer
	mIdxBuilder := new(test.MockIndexBuilder)
	mIdx := new(test.MockIndexer)
	mIdxBuilder.MockIndexer = mIdx

	// fake storage and mock exporter
	fStorage := new(test.FakeStorage)
	mExporter := new(test.MockExporter)

	// create new index manager with mocks values
	manager, err := NewManager(fStorage, mIdxBuilder, t.Name(), WithExporter(mExporter))
	require.NoError(t, err)

	// a fake tipset to index
	tsHeight := int64(1)
	ts1 := test.MustFakeTipSet(t, tsHeight)

	// results channel and error channel MockIndexer will return.
	resChan := make(chan *tipset.Result, 1)
	errChan := make(chan error, 1)

	// expect index manager to pass MockedIndexer anything (ctx) and the tipset, returning the channels created above.
	mIdxBuilder.MockIndexer.On("TipSet", mock.Anything, ts1).Return(resChan, errChan, nil)

	// create some fake data and a processing report
	data := &test.FakePersistable{}
	report := visormodel.ProcessingReportList{
		&visormodel.ProcessingReport{
			Height:      tsHeight,
			StateRoot:   "stateroot",
			Reporter:    t.Name(),
			Task:        "task",
			StartedAt:   time.Unix(0, 0),
			CompletedAt: time.Unix(0, 0),
			// status of OK means indexing was successful
			Status:            visormodel.ProcessingStatusOK,
			StatusInformation: "",
			ErrorsDetected:    nil,
		},
	}
	// send report to index manager and close the channel to signal MockIndexer is done indexing data.
	resChan <- &tipset.Result{
		Name:   t.Name(),
		Data:   data,
		Report: report,
	}

	// send a fatal error to the index manager
	errChan <- fmt.Errorf("fatal error")
	close(resChan)
	close(errChan)

	// mock exporter expects to receieve a result with data and report
	mExporter.On("ExportResult", mock.Anything, fStorage, int64(ts1.Height()), []*indexer.ModelResult{
		{
			Name:  t.Name(),
			Model: model.PersistableList{report, data},
		},
	}).Return(nil)

	success, err := manager.TipSet(ctx, ts1)
	require.Error(t, err)
	require.False(t, success)
}

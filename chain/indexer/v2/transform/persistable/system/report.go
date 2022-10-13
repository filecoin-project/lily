package system

import (
	"context"
	"reflect"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	v2 "github.com/filecoin-project/lily/model/v2"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

var log = logging.Logger("transform/report")

type ProcessingReportTransform struct {
	meta v2.ModelMeta
}

func NewProcessingReportTransform() *ProcessingReportTransform {
	return &ProcessingReportTransform{}
}

func (s *ProcessingReportTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	results := make(map[v2.ModelMeta][]*extract.StateResult)
	sqlModels := make(visormodel.ProcessingReportList, 0, 16)
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			results[res.Task()] = append(results[res.Task()], res.State())
		}
	}
	for task, states := range results {
		status := visormodel.ProcessingStatusOK
		firstStart := time.Unix(999999999999, 0)
		longestDuration := time.Nanosecond
		var errs []error
		var height int64
		var stateroot string
		for _, state := range states {
			if len(state.Data) > 0 {
				height = int64(state.Data[0].ChainEpochTime().Height)
				stateroot = state.Data[0].ChainEpochTime().StateRoot.String()
			}
			if state.StartedAt.Before(firstStart) {
				firstStart = state.StartedAt
			}
			if state.Duration > longestDuration {
				longestDuration = state.Duration
			}
			if state.Error != nil {
				status = visormodel.ProcessingStatusError
				errs = append(errs, state.Error)
			}
		}
		sqlModels = append(sqlModels, &visormodel.ProcessingReport{
			Height:            height,
			StateRoot:         stateroot,
			Reporter:          "TODO",
			Task:              task.String(), // TODO need a revers look up from Meta to task name
			StartedAt:         firstStart,
			CompletedAt:       firstStart.Add(longestDuration),
			Status:            status,
			StatusInformation: "",
			ErrorsDetected:    errs,
		})
	}
	if len(sqlModels) > 0 {
		out <- &persistable.Result{Model: sqlModels}
	}
	return nil
}

func (s *ProcessingReportTransform) ModelType() v2.ModelMeta {
	return v2.ModelMeta{}
}

func (s *ProcessingReportTransform) Name() string {
	info := ProcessingReportTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *ProcessingReportTransform) Matcher() string {
	return ".*"
}

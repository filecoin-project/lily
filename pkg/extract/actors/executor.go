package actors

import (
	"context"
	"time"

	"github.com/filecoin-project/lily/tasks"
)

type DifferReport struct {
	DiffType  string
	StartTime time.Time
	Duration  time.Duration
	Error     error
	Result    ActorStateChange
}

func ExecuteStateDiff(ctx context.Context, api tasks.DataSource, act *ActorChange, fns ...ActorDiffMethods) []DifferReport {
	out := make([]DifferReport, len(fns))
	for i, fn := range fns {
		start := time.Now()
		res, err := fn.Diff(ctx, api, act)
		out[i] = DifferReport{
			DiffType:  fn.Type(),
			StartTime: start,
			Duration:  time.Since(start),
			Error:     err,
			Result:    res,
		}
	}
	return out
}

type StateDiffer struct {
	Methods       []ActorDiffMethods
	ReportHandler ReportHandlerFn
	ActorHandler  ActorHandlerFn
}

type ReportHandlerFn = func(reports []DifferReport) error
type ActorHandlerFn = func(changes []ActorStateChange) (ActorDiffResult, error)

func (s *StateDiffer) ActorDiff(ctx context.Context, api tasks.DataSource, act *ActorChange) (ActorDiffResult, error) {
	reports := ExecuteStateDiff(ctx, api, act, s.Methods...)

	if s.ReportHandler != nil {
		if err := s.ReportHandler(reports); err != nil {
			return nil, err
		}
	}

	var results []ActorStateChange
	for _, report := range reports {
		results = append(results, report.Result)
	}
	return s.ActorHandler(results)
}

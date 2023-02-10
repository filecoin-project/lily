package actors

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/extract/actors")

type DifferReport struct {
	DiffType  string
	StartTime time.Time
	Duration  time.Duration
	Result    ActorStateChange
}

type StateDiffer struct {
	Methods       []ActorDiffMethods
	ReportHandler ReportHandlerFn
	ActorHandler  ActorHandlerFn
}

type ReportHandlerFn = func(reports []DifferReport) error
type ActorHandlerFn = func(changes []ActorStateChange) (DiffResult, error)

func (s *StateDiffer) ActorDiff(ctx context.Context, api tasks.DataSource, act *Change) (DiffResult, error) {
	out := make([]DifferReport, len(s.Methods))
	for i, fn := range s.Methods {
		start := time.Now()
		res, err := fn.Diff(ctx, api, act)
		if err != nil {
			return nil, err
		}
		out[i] = DifferReport{
			DiffType:  fn.Type(),
			StartTime: start,
			Duration:  time.Since(start),
			Result:    res,
		}
	}

	if s.ReportHandler != nil {
		if err := s.ReportHandler(out); err != nil {
			return nil, err
		}
	}

	var results []ActorStateChange
	for _, report := range out {
		results = append(results, report.Result)
	}
	return s.ActorHandler(results)
}

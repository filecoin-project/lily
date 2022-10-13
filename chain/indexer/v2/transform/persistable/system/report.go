package system

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/filecoin-project/go-address"
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
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			switch v := res.ExtractionState().(type) {
			case *extract.ActorStateResult:
				var errors []interface{}
				var slowestActor address.Address
				var longestDuration time.Duration
				status := visormodel.ProcessingStatusOK
				for _, p := range v.Results {
					if p.Error != nil {
						errors = append(errors, p.Error)
						status = visormodel.ProcessingStatusError
					}
					if p.Duration > longestDuration {
						longestDuration = p.Duration
						slowestActor = p.Info.Address
					}
				}
				report := &visormodel.ProcessingReport{
					Height:            int64(v.TipSet.Height()),
					StateRoot:         v.TipSet.ParentState().String(),
					Reporter:          "TODO",
					Task:              v.Task.String(),
					StartedAt:         v.StartTime,
					CompletedAt:       v.StartTime.Add(v.Duration),
					Status:            status,
					StatusInformation: fmt.Sprintf("slowest actor: %s duration: %s", slowestActor, longestDuration),
				}
				if len(errors) > 0 {
					report.ErrorsDetected = errors
				}
				out <- &persistable.Result{Model: report}
			case *extract.TipSetStateResult:
				report := &visormodel.ProcessingReport{
					Height:      int64(v.TipSet.Height()),
					StateRoot:   v.TipSet.ParentState().String(),
					Reporter:    "TODO",
					Task:        v.Task.String(),
					StartedAt:   v.StartTime,
					CompletedAt: v.StartTime.Add(v.Duration),
				}
				report.Status = visormodel.ProcessingStatusOK
				if v.Error != nil && v.Error.Error != nil {
					report.Status = visormodel.ProcessingStatusError
					report.ErrorsDetected = v.Error
				}
				out <- &persistable.Result{Model: report}
			}
		}
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

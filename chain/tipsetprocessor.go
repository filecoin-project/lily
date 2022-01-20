package chain

import (
	"context"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lotus/chain/types"
	"time"
)

var _ TipSetStateExtractorProcessor = (*TipSetExtractorProcessorImpl)(nil)

type TipSetExtractorProcessorImpl struct {
	api               lens.API
	current, previous *types.TipSet
}

func NewTipSetExtractorProcessorImpl(api lens.API, current, previous *types.TipSet) *TipSetExtractorProcessorImpl {
	return &TipSetExtractorProcessorImpl{
		api:      api,
		current:  current,
		previous: previous,
	}
}

func (t TipSetExtractorProcessorImpl) ProcessTipSetExtractors(ctx context.Context, current *types.TipSet, previous *types.TipSet, extractor TipSetStateExtractor) (model.Persistable, *visormodel.ProcessingReport, error) {
	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
		Status:    visormodel.ProcessingStatusOK,
	}

	start := time.Now()
	data, err := extractor.Extract(ctx, current, previous, t.api)
	log.Infow("tiptset extracted", "name", extractor.Name(), "current", current.Height(), "previous", previous.Height(), "duration", time.Since(start).String())
	return data, report, err
}

func (t TipSetExtractorProcessorImpl) Run(ctx context.Context, extractor TipSetStateExtractor, results chan *TaskResult) {
	start := time.Now()

	data, report, err := t.ProcessTipSetExtractors(ctx, t.current, t.previous, extractor)
	if err != nil {
		results <- &TaskResult{
			Task:        extractor.Name(),
			Error:       err,
			StartedAt:   start,
			CompletedAt: time.Now(),
		}
		return
	}
	results <- &TaskResult{
		Task:        extractor.Name(),
		Report:      visormodel.ProcessingReportList{report},
		Data:        data,
		StartedAt:   start,
		CompletedAt: time.Now(),
	}
}

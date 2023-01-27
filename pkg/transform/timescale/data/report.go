package data

import (
	"time"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

type ReportError struct {
	Error string
}

type ProcessingReportBuilder struct {
	errors []*ReportError
	models model.PersistableList
	report *visormodel.ProcessingReport
}

func (b *ProcessingReportBuilder) WithStatus(status string) *ProcessingReportBuilder {
	b.report.Status = status
	return b
}

func (b *ProcessingReportBuilder) WithInformation(info string) *ProcessingReportBuilder {
	b.report.StatusInformation = info
	return b
}

func (b *ProcessingReportBuilder) AddError(err error) *ProcessingReportBuilder {
	b.errors = append(b.errors, &ReportError{Error: err.Error()})
	return b
}

func (b *ProcessingReportBuilder) AddModels(m ...model.Persistable) *ProcessingReportBuilder {
	b.models = append(b.models, m...)
	return b
}

func (b *ProcessingReportBuilder) Finish() model.PersistableList {
	b.report.CompletedAt = time.Now()
	if len(b.errors) == 0 {
		if b.report.Status == "" {
			b.WithStatus(visormodel.ProcessingStatusOK)
		}
	} else {
		b.WithStatus(visormodel.ProcessingStatusError)
	}
	return model.PersistableList{b.models, b.report}
}

func StartProcessingReport(task string, ts *types.TipSet) *ProcessingReportBuilder {
	return &ProcessingReportBuilder{
		report: &visormodel.ProcessingReport{
			Height:    int64(ts.Height()),
			StateRoot: ts.ParentState().String(),
			Reporter:  "Deprecate",
			Task:      task,
			StartedAt: time.Now(),
		},
	}
}

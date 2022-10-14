package chain

import (
	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

func ToProcessingReport(task string, reporter string, res *extract.TipSetStateResult) *visormodel.ProcessingReport {
	status := visormodel.ProcessingStatusOK
	if res.Error != nil {
		status = visormodel.ProcessingStatusError
	}
	return &visormodel.ProcessingReport{
		Height:            int64(res.TipSet.Height()),
		StateRoot:         res.TipSet.ParentState().String(),
		Reporter:          reporter,
		Task:              task,
		StartedAt:         res.StartTime,
		CompletedAt:       res.StartTime.Add(res.Duration),
		Status:            status,
		StatusInformation: "",
		ErrorsDetected:    res.Error,
	}
}

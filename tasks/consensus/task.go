package consensus

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/chain"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
)

type Task struct{}

func NewTask() *Task {
	return &Task{}
}

func (t *Task) ProcessTipSets(ctx context.Context, child, parent *types.TipSet) (model.Persistable, visormodel.ProcessingReportList, error) {
	_, span := global.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(label.String("child", child.String()), label.Int64("height", int64(child.Height())))
		span.SetAttributes(label.String("parent", parent.String()), label.Int64("height", int64(parent.Height())))
	}
	defer span.End()

	pl := make(chain.ChainConsensusList, child.Height()-parent.Height())
	rp := make(visormodel.ProcessingReportList, child.Height()-parent.Height())
	idx := 0
	for epoch := parent.Height(); epoch < child.Height(); epoch++ {
		if parent.Height() == epoch {
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: parent.ParentState().String(),
				ParentTipSet:    parent.Parents().String(),
				TipSet:          parent.Key().String(),
			}
			rp[idx] = &visormodel.ProcessingReport{
				Height:    int64(epoch),
				StateRoot: parent.ParentState().String(),
			}
		} else {
			// null round no tipset
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: parent.ParentState().String(),
				ParentTipSet:    parent.Parents().String(),
				TipSet:          "",
			}
			rp[idx] = &visormodel.ProcessingReport{
				Height:            int64(epoch),
				StateRoot:         parent.ParentState().String(),
				StatusInformation: "Null Round",
			}
		}
		idx += 1
	}
	return pl, rp, nil
}

func (t *Task) Close() error {
	return nil
}

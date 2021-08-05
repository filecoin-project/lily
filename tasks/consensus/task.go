package consensus

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/consensus"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
)

type Task struct {
}

func NewTask() *Task {
	return &Task{}
}

func (t *Task) ProcessTipSets(ctx context.Context, child, parent *types.TipSet) (model.Persistable, visormodel.ProcessingReportList, error) {
	_, span := global.Tracer("").Start(ctx, "ProcessBlocks")
	if span.IsRecording() {
		span.SetAttributes(label.String("child", child.String()), label.Int64("height", int64(child.Height())))
		span.SetAttributes(label.String("parent", parent.String()), label.Int64("height", int64(parent.Height())))
	}
	defer span.End()

	var pl consensus.ChainConsensusList
	var rp visormodel.ProcessingReportList
	for epoch := child.Height(); epoch > parent.Height(); epoch-- {
		if child.Height() == epoch {
			pl = append(pl, &consensus.ChainConsensus{
				Height:       int64(epoch),
				StateRoot:    child.ParentState().String(),
				ParentTipSet: child.Parents().String(),
				TipSet:       child.Key().String(),
			})
			rp = append(rp, &visormodel.ProcessingReport{
				Height:    int64(epoch),
				StateRoot: child.ParentState().String(),
			})
		} else {
			// null round no tipset
			pl = append(pl, &consensus.ChainConsensus{
				Height:       int64(epoch),
				StateRoot:    parent.ParentState().String(),
				ParentTipSet: parent.Parents().String(),
				TipSet:       "",
			})
			rp = append(rp, &visormodel.ProcessingReport{
				Height:            int64(epoch),
				StateRoot:         parent.ParentState().String(),
				StatusInformation: "Null Round",
			})
		}
	}
	return pl, rp, nil
}

func (t *Task) Close() error {
	return nil
}

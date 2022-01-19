package consensus

import (
	"context"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/chain"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Task struct{}

func NewTask() *Task {
	return &Task{}
}

func (t *Task) ProcessTipSets(ctx context.Context, current, previous *types.TipSet) (model.Persistable, visormodel.ProcessingReportList, error) {
	_, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("current", current.String()), attribute.Int64("height", int64(current.Height())))
		span.SetAttributes(attribute.String("previous", previous.String()), attribute.Int64("height", int64(previous.Height())))
	}
	defer span.End()

	pl := make(chain.ChainConsensusList, current.Height()-previous.Height())
	rp := make(visormodel.ProcessingReportList, current.Height()-previous.Height())
	idx := 0
	// walk from head to the previous
	for epoch := current.Height(); epoch > previous.Height(); epoch-- {
		if current.Height() == epoch {
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: current.ParentState().String(),
				ParentTipSet:    current.Parents().String(),
				TipSet:          current.Key().String(),
			}
			rp[idx] = &visormodel.ProcessingReport{
				Height:    int64(epoch),
				StateRoot: current.ParentState().String(),
			}
		} else {
			// null round no tipset
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: current.ParentState().String(),
				ParentTipSet:    current.Parents().String(),
				TipSet:          "",
			}
			rp[idx] = &visormodel.ProcessingReport{
				Height:            int64(epoch),
				StateRoot:         current.ParentState().String(),
				StatusInformation: visormodel.ProcessingStatusInformationNullRound,
			}
		}
		idx += 1
	}
	return pl, rp, nil
}

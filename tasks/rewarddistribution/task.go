package rewarddistribution

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/chain/types"
)

const unsupportedInformation = "reward distribution requires network version 29"

// Task records the block reward distribution produced by executing a tipset.
type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{node: node}
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "chain_reward_distributions"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	distribution, err := t.node.RewardDistribution(ctx, executed)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting reward distribution: %w", err)
		return nil, report, nil
	}
	if distribution == nil {
		report.StatusInformation = unsupportedInformation
		return nil, report, nil
	}

	payload, err := json.Marshal(distribution)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("marshaling reward distribution: %w", err)
		return nil, report, nil
	}

	return &rewardmodel.ChainRewardDistribution{
		Height:       int64(executed.Height()),
		TipSetKey:    executed.Key().String(),
		Distribution: string(payload),
	}, report, nil
}

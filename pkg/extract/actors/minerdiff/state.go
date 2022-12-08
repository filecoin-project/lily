package minerdiff

import (
	"context"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiff struct {
	InfoChange          *InfoChange
	FundsChange         *FundsChange
	DebtChange          *DebtChange
	PreCommitChanges    PreCommitChangeList
	SectorChanges       SectorChangeList
	SectorStatusChanges *SectorStatusChange
}

type ActorStateKind string

func State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, diffFns ...actors.ActorDiffer) (*StateDiff, error) {
	var stateDiff = new(StateDiff)
	for _, f := range diffFns {
		// TODO maybe this method should also return a bool to indicate if anything actually changed, instead of two null values.
		stateChange, err := f.Diff(ctx, api, act)
		if err != nil {
			return nil, err
		}
		if stateChange == nil {
			continue
		}
		switch stateChange.Kind() {
		case KindMinerInfo:
			stateDiff.InfoChange = stateChange.(*InfoChange)
		case KindMinerSector:
			stateDiff.SectorChanges = stateChange.(SectorChangeList)
		case KindMinerPreCommit:
			stateDiff.PreCommitChanges = stateChange.(PreCommitChangeList)
		case KindMinerFunds:
			stateDiff.FundsChange = stateChange.(*FundsChange)
		case KindMinerDebt:
			stateDiff.DebtChange = stateChange.(*DebtChange)
		case KindMinerSectorStatus:
			stateDiff.SectorStatusChanges = stateChange.(*SectorStatusChange)
		}
	}
	return stateDiff, nil
}

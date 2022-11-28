package minerdiff

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiff struct {
	TipSet           *types.TipSet
	Miner            *core.ActorChange
	InfoChange       *InfoChange
	SectorChanges    SectorChangeList
	PreCommitChanges PreCommitChangeList
}

type ActorStateKind string

type ActorStateChange interface {
	Kind() ActorStateKind
}

type ActorDiffer interface {
	Diff(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet) (ActorStateChange, error)
}

func State(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet, diffFns ...ActorDiffer) (*StateDiff, error) {
	stateDiff := &StateDiff{
		TipSet: current,
		Miner:  act,
	}
	for _, f := range diffFns {
		stateChange, err := f.Diff(ctx, api, act, current, executed)
		if err != nil {
			return nil, err
		}
		switch stateChange.Kind() {
		case "miner_info":
			stateDiff.InfoChange = stateChange.(*InfoChange)
		case "miner_sector":
			stateDiff.SectorChanges = stateChange.(SectorChangeList)
		case "miner_precommit":
			stateDiff.PreCommitChanges = stateChange.(PreCommitChangeList)
		}
	}
	panic("here")
}

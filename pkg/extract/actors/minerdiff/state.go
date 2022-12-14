package minerdiff

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/miner")

type StateDiff struct {
	InfoChange          *InfoChange
	FundsChange         *FundsChange
	DebtChange          *DebtChange
	PreCommitChanges    PreCommitChangeList
	SectorChanges       SectorChangeList
	SectorStatusChanges *SectorStatusChange
}

func (s *StateDiff) Kind() string {
	return "miner"
}

func State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, diffFns ...actors.ActorDiffer) (actors.ActorStateDiff, error) {
	grp, grpCtx := errgroup.WithContext(ctx)
	results := make(chan actors.ActorStateChange, len(diffFns))

	for _, f := range diffFns {
		f := f
		grp.Go(func() error {
			stateChange, err := f.Diff(grpCtx, api, act)
			if err != nil {
				return err
			}

			// TODO maybe this method should also return a bool to indicate if anything actually changed, instead of two null values.
			if stateChange != nil {
				results <- stateChange
			}
			return nil
		})
	}

	go func() {
		if err := grp.Wait(); err != nil {
			log.Error(err)
		}
		close(results)
	}()
	var stateDiff = new(StateDiff)
	for stateChange := range results {
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

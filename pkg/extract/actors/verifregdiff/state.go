package verifregdiff

import (
	"context"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiff struct {
	VerifierChanges VerifiersChangeList
	ClientChanges   ClientsChangeList
	ClaimChanges    ClaimsChangeList
}

func State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, diffFns ...actors.ActorDiffer) (*StateDiff, error) {
	var stateDiff = new(StateDiff)
	for _, f := range diffFns {
		stateChange, err := f.Diff(ctx, api, act)
		if err != nil {
			return nil, err
		}
		if stateChange == nil {
			continue
		}
		switch stateChange.Kind() {
		case KindVerifregClients:
			stateDiff.ClientChanges = stateChange.(ClientsChangeList)
		case KindVerifregClaims:
			stateDiff.ClaimChanges = stateChange.(ClaimsChangeList)
		case KindVerifregVerifiers:
			stateDiff.VerifierChanges = stateChange.(VerifiersChangeList)
		}
	}
	return stateDiff, nil
}

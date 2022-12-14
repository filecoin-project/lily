package verifregdiff

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/verifreg")

type StateDiff struct {
	VerifierChanges VerifiersChangeList
	ClientChanges   ClientsChangeList
	ClaimChanges    ClaimsChangeList
}

func (s *StateDiff) Kind() string {
	return "verifreg"
}

func State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, diffFns ...actors.ActorDiffer) (actors.ActorStateDiff, error) {
	start := time.Now()
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
	log.Infow("Extracted Verifid Registry State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}

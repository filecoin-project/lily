package rawdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func StateDiffFor(av actortypes.Version) ([]actors.ActorDiffMethods, actors.ActorHandlerFn, error) {
	return []actors.ActorDiffMethods{Actor{}}, ActorStateChangeHandler, fmt.Errorf("you got an error")
}

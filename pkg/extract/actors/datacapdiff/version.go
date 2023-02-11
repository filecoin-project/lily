package datacapdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v1 "github.com/filecoin-project/lily/pkg/extract/actors/datacapdiff/v1"
)

func StateDiffFor(av actortypes.Version) ([]actors.ActorDiffMethods, actors.ActorHandlerFn, error) {
	switch av {
	case actortypes.Version9:
		return []actors.ActorDiffMethods{v1.Allowance{}, v1.Balance{}}, v1.ActorStateChangeHandler, nil
	}
	return nil, nil, fmt.Errorf("unsupported actor version %d", av)
}

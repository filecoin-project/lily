package verifregdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v1 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v1"
	v2 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v2"
)

func StateDiffFor(av actortypes.Version) ([]actors.ActorDiffMethods, actors.ActorHandlerFn, error) {
	if av < actortypes.Version9 {
		return []actors.ActorDiffMethods{
			v1.Clients{},
			v1.Verifiers{},
		}, v1.ActorStateChangeHandler, nil
	} else if av == actortypes.Version9 {
		return []actors.ActorDiffMethods{
			v2.Verifiers{},
			v2.Claims{},
			v2.Allocations{},
		}, v2.ActorStateChangeHandler, nil
	}
	return nil, nil, fmt.Errorf("unsupported actor version %d", av)
}

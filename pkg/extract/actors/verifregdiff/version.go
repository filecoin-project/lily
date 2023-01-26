package verifregdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v1 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v1"
	v2 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v2"
)

func StateDiffFor(av actortypes.Version) (actors.ActorDiff, error) {
	if av < actortypes.Version9 {
		return &v1.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v1.Clients{},
				v1.Verifiers{},
			}}, nil
	} else if av == actortypes.Version9 {
		return &v2.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v2.Verifiers{},
				v2.Claims{},
				v2.Allocations{},
			}}, nil
	}
	return nil, fmt.Errorf("unsupported actor version %d", av)
}

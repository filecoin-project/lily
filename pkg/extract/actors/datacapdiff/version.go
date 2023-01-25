package datacapdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v9 "github.com/filecoin-project/lily/pkg/extract/actors/datacapdiff/v9"
)

func StateDiffFor(av actortypes.Version) (actors.ActorDiff, error) {
	switch av {
	case actortypes.Version9:
		return &v9.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v9.Allowance{},
				v9.Balance{},
			},
		}, nil
	case actortypes.Version10:
		panic("Not yet implemented")
	}
	return nil, fmt.Errorf("unsupported actor version %d", av)
}

package marketdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v1 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v1"
)

func StateDiffFor(av actortypes.Version) ([]actors.ActorDiffMethods, actors.ActorHandlerFn, error) {
	switch av {
	case actortypes.Version0, actortypes.Version2, actortypes.Version3, actortypes.Version4, actortypes.Version5,
		actortypes.Version6, actortypes.Version7, actortypes.Version8, actortypes.Version9:
		return []actors.ActorDiffMethods{
			v1.Deals{},
			v1.Proposals{},
		}, v1.ActorStateChangeHandler, nil
	}
	return nil, nil, fmt.Errorf("unsupported actor version %d", av)
}

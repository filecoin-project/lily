package minerdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v1 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"
)

func StateDiffFor(av actortypes.Version) (actors.ActorDiff, error) {
	switch av {
	case actortypes.Version0, actortypes.Version2, actortypes.Version3, actortypes.Version4, actortypes.Version5,
		actortypes.Version6, actortypes.Version7, actortypes.Version8, actortypes.Version9:
		return &v1.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v1.Info{},
				v1.PreCommit{},
				v1.SectorStatus{},
				v1.Sectors{},
			}}, nil
	case actortypes.Version10:
		panic("Not yet implemented")
	}
	return nil, fmt.Errorf("unsupported actor version %d", av)
}

package minerdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
	v2 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v2"
	v3 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v3"
	v4 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v4"
	v5 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v5"
	v6 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v6"
	v7 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v7"
	v8 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v8"
	v9 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v9"
)

func StateDiffFor(av actortypes.Version) (actors.ActorDiff, error) {
	switch av {
	case actortypes.Version0:
		return &v0.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v0.Info{},
				v0.PreCommit{},
				v0.SectorStatus{},
				v0.Sectors{},
			}}, nil
	case actortypes.Version2:
		return &v2.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v2.Info{},
				v2.PreCommit{},
				v2.SectorStatus{},
				v2.Sectors{},
			}}, nil
	case actortypes.Version3:
		return &v3.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v3.Info{},
				v3.PreCommit{},
				v3.SectorStatus{},
				v3.Sectors{},
			}}, nil
	case actortypes.Version4:
		return &v4.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v4.Info{},
				v4.PreCommit{},
				v4.SectorStatus{},
				v4.Sectors{},
			}}, nil
	case actortypes.Version5:
		return &v5.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v5.Info{},
				v5.PreCommit{},
				v5.SectorStatus{},
				v5.Sectors{},
			}}, nil
	case actortypes.Version6:
		return &v6.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v6.Info{},
				v6.PreCommit{},
				v6.SectorStatus{},
				v6.Sectors{},
			}}, nil
	case actortypes.Version7:
		return &v7.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v7.Info{},
				v7.PreCommit{},
				v7.SectorStatus{},
				v7.Sectors{},
			}}, nil
	case actortypes.Version8:
		return &v8.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v8.Info{},
				v8.PreCommit{},
				v8.SectorStatus{},
				v8.Sectors{},
			}}, nil
	case actortypes.Version9:
		return &v9.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v9.Info{},
				v9.PreCommit{},
				v9.SectorStatus{},
				v9.Sectors{},
			}}, nil
	case actortypes.Version10:
		panic("Not yet implemented")
	}
	return nil, fmt.Errorf("unsupported actor version %d", av)
}

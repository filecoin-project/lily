package verifregdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
	v2 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v2"
	v3 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v3"
	v4 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v4"
	v5 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v5"
	v6 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v6"
	v7 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v7"
	v8 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v8"
	v9 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v9"
)

func StateDiffFor(av actortypes.Version) (actors.ActorDiff, error) {
	switch av {
	case actortypes.Version0:
		return &v0.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v0.Clients{},
				v0.Verifiers{},
			}}, nil
	case actortypes.Version2:
		return &v2.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v2.Clients{},
				v2.Verifiers{},
			}}, nil
	case actortypes.Version3:
		return &v3.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v3.Clients{},
				v3.Verifiers{},
			}}, nil
	case actortypes.Version4:
		return &v4.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v4.Clients{},
				v4.Verifiers{},
			}}, nil
	case actortypes.Version5:
		return &v5.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v5.Clients{},
				v5.Verifiers{},
			}}, nil
	case actortypes.Version6:
		return &v6.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v6.Clients{},
				v6.Verifiers{},
			}}, nil
	case actortypes.Version7:
		return &v7.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v7.Clients{},
				v7.Verifiers{},
			}}, nil
	case actortypes.Version8:
		return &v8.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v8.Clients{},
				v8.Verifiers{},
			}}, nil
	case actortypes.Version9:
		return &v9.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v9.Verifiers{},
				v9.Claims{},
				// TODO implement this
				//v9.Allocations{},
			}}, nil
	case actortypes.Version10:
		panic("Not yet implemented")
	}
	return nil, fmt.Errorf("unsupported actor version %d", av)
}

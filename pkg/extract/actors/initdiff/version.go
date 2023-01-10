package initdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v0"
	v2 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v2"
	v3 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v3"
	v4 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v4"
	v5 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v5"
	v6 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v6"
	v7 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v7"
	v8 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v8"
	v9 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v9"
)

func StateDiffFor(av actortypes.Version) (actors.ActorDiff, error) {
	switch av {
	case actortypes.Version0:
		return &v0.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v0.Addresses{},
			}}, nil
	case actortypes.Version2:
		return &v2.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v2.Addresses{},
			}}, nil
	case actortypes.Version3:
		return &v3.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v3.Addresses{},
			}}, nil
	case actortypes.Version4:
		return &v4.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v4.Addresses{},
			}}, nil
	case actortypes.Version5:
		return &v5.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v5.Addresses{},
			}}, nil
	case actortypes.Version6:
		return &v6.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v6.Addresses{},
			}}, nil
	case actortypes.Version7:
		return &v7.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v7.Addresses{},
			}}, nil
	case actortypes.Version8:
		return &v8.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v8.Addresses{},
			}}, nil
	case actortypes.Version9:
		return &v9.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v9.Addresses{},
			}}, nil
	case actortypes.Version10:
		panic("NYI")
	}
	return nil, fmt.Errorf("unsupported actor version %d", av)
}

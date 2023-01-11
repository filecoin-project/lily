package marketdiff

import (
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v0"
	v2 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v2"
	v3 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v3"
	v4 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v4"
	v5 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v5"
	v6 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v6"
	v7 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v7"
	v8 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v8"
	v9 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v9"
)

func StateDiffFor(av actortypes.Version) (actors.ActorDiff, error) {
	switch av {
	case actortypes.Version0:
		return &v0.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v0.Deals{},
				v0.Proposals{},
			}}, nil
	case actortypes.Version2:
		return &v2.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v2.Deals{},
				v2.Proposals{},
			}}, nil
	case actortypes.Version3:
		return &v3.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v3.Deals{},
				v3.Proposals{},
			}}, nil
	case actortypes.Version4:
		return &v4.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v4.Deals{},
				v4.Proposals{},
			}}, nil
	case actortypes.Version5:
		return &v5.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v5.Deals{},
				v5.Proposals{},
			}}, nil
	case actortypes.Version6:
		return &v6.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v6.Deals{},
				v6.Proposals{},
			}}, nil
	case actortypes.Version7:
		return &v7.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v7.Deals{},
				v7.Proposals{},
			}}, nil
	case actortypes.Version8:
		return &v8.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v8.Deals{},
				v8.Proposals{},
			}}, nil
	case actortypes.Version9:
		return &v8.StateDiff{
			DiffMethods: []actors.ActorStateDiff{
				v9.Deals{},
				v9.Proposals{},
			}}, nil
	case actortypes.Version10:
		panic("NYI")
	}
	return nil, fmt.Errorf("unsupported actor version %d", av)
}

package processor

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/transform/persistable/minertransform"
	"github.com/filecoin-project/lily/tasks"
)

func Diff(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet) {
	minerStateDiff, err := minerdiff.State(ctx, api, act, current, executed,
		minerdiff.Info{},
		minerdiff.PreCommit{},
		minerdiff.Sectors{},
	)
	if err != nil {
		panic(err)
	}
	models, err := minertransform.State(ctx, minerStateDiff)
	if err != nil {
		panic(err)
	}
}

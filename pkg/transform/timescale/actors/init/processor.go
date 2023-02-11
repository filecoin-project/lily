package init

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	initdiff "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v1"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

func TransformInitState(ctx context.Context, s store.Store, current, executed *types.TipSet, initMapRoot *cid.Cid) (model.Persistable, error) {
	if initMapRoot == nil {
		return data.StartProcessingReport(tasktype.IDAddress, current).
			WithStatus(visormodel.ProcessingStatusInfo).
			WithInformation("no change detected").
			Finish(), nil
	}
	initState := new(initdiff.StateChange)
	if err := s.Get(ctx, *initMapRoot, initState); err != nil {
		return nil, err
	}

	initStateDiff, err := initState.ToStateDiffResult(ctx, s)
	if err != nil {
		return nil, err
	}

	return Addresses{}.Extract(ctx, current, executed, initStateDiff), nil
}

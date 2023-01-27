package verifreg

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	v1 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1"
	v2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v2"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

func TransformVerifregState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root *cid.Cid) (model.Persistable, error) {
	if root == nil {
		if version < actortypes.Version9 {
			return model.PersistableList{
				data.StartProcessingReport(tasktype.VerifiedRegistryVerifiedClient, current).
					WithStatus(visormodel.ProcessingStatusInfo).
					WithInformation("no change detected").
					Finish(),
				data.StartProcessingReport(tasktype.VerifiedRegistryVerifier, current).
					WithStatus(visormodel.ProcessingStatusInfo).
					WithInformation("no change detected").
					Finish(),
			}, nil
		}
		return model.PersistableList{
			data.StartProcessingReport("verified_registry_claims", current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport("verified_registry_allocation", current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport(tasktype.VerifiedRegistryVerifier, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
		}, nil
	}
	switch version {
	case actortypes.Version0,
		actortypes.Version2,
		actortypes.Version3,
		actortypes.Version4,
		actortypes.Version5,
		actortypes.Version6,
		actortypes.Version7,
		actortypes.Version8:
		return v1.TransformVerifregState(ctx, s, version, current, executed, *root)

	case actortypes.Version9:
		return v2.TransformVerifregState(ctx, s, version, current, executed, *root)
	}
	return nil, fmt.Errorf("unsupported version : %d", version)

}

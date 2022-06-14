package ipfs

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

type IpfsSyncTask struct {
}

func (i *IpfsSyncTask) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	current.ParentState()
}

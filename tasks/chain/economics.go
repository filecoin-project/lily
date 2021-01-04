package chain

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model"
	chainmodel "github.com/filecoin-project/sentinel-visor/model/chain"
	"github.com/filecoin-project/sentinel-visor/model/visor"
)

const (
	idleSleepInterval = 60 * time.Second       // time to wait if the processor runs out of blocks to process
	batchInterval     = 100 * time.Millisecond // time to wait between batches
)

var log = logging.Logger("chain")

type EconomicsStorage interface {
	PersistBatch(ctx context.Context, ps ...model.Persistable) error
	MarkTipSetEconomicsComplete(ctx context.Context, tipset string, height int64, completedAt time.Time, errorsDetected string) error
	LeaseTipSetEconomics(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingTipSetList, error)
}

type ChainEconomicsLens interface {
	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (api.CirculatingSupply, error)
}

func ExtractChainEconomicsModel(ctx context.Context, node ChainEconomicsLens, ts *types.TipSet) (*chainmodel.ChainEconomics, error) {
	supply, err := node.StateVMCirculatingSupplyInternal(ctx, ts.Key())
	if err != nil {
		return nil, xerrors.Errorf("get circulating supply: %w", err)
	}

	return &chainmodel.ChainEconomics{
		ParentStateRoot: ts.ParentState().String(),
		VestedFil:       supply.FilVested.String(),
		MinedFil:        supply.FilMined.String(),
		BurntFil:        supply.FilBurnt.String(),
		LockedFil:       supply.FilLocked.String(),
		CirculatingFil:  supply.FilCirculating.String(),
	}, nil
}

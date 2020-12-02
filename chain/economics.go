package chain

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	chainmodel "github.com/filecoin-project/sentinel-visor/model/chain"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

type ChainEconomicsProcessor struct {
	node   lens.API
	opener lens.APIOpener
	closer lens.APICloser
}

func NewChainEconomicsProcessor(opener lens.APIOpener) *ChainEconomicsProcessor {
	return &ChainEconomicsProcessor{
		opener: opener,
	}
}

func (p *ChainEconomicsProcessor) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.PersistableWithTx, *visormodel.ProcessingReport, error) {
	if p.node == nil {
		node, closer, err := p.opener.Open(ctx)
		if err != nil {
			return nil, nil, xerrors.Errorf("unable to open lens: %w", err)
		}
		p.node = node
		p.closer = closer
	}
	// TODO: close lens if rpc error

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	supply, err := p.node.StateVMCirculatingSupplyInternal(ctx, ts.Key())
	if err != nil {
		log.Errorw("error received while fetching circulating supply messages, closing lens", "error", err)
		if cerr := p.Close(); cerr != nil {
			log.Errorw("error received while closing lens", "error", cerr)
		}
		return nil, nil, err
	}

	ce := &chainmodel.ChainEconomics{
		ParentStateRoot: ts.ParentState().String(),
		VestedFil:       supply.FilVested.String(),
		MinedFil:        supply.FilMined.String(),
		BurntFil:        supply.FilBurnt.String(),
		LockedFil:       supply.FilLocked.String(),
		CirculatingFil:  supply.FilCirculating.String(),
	}

	return ce, report, nil
}

func (p *ChainEconomicsProcessor) Close() error {
	if p.closer != nil {
		p.closer()
		p.closer = nil
	}
	p.node = nil
	return nil
}

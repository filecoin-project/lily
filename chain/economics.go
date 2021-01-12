package chain

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/tasks/chain"
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

func (p *ChainEconomicsProcessor) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
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

	ce, err := chain.ExtractChainEconomicsModel(ctx, p.node, ts)
	if err != nil {
		log.Errorw("error received while extracting chain economics, closing lens", "error", err)
		if cerr := p.Close(); cerr != nil {
			log.Errorw("error received while closing lens", "error", cerr)
		}
		return nil, nil, err
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

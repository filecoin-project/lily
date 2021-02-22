package chaineconomics

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

var log = logging.Logger("chaineconomics")

type Task struct {
	node   lens.API
	opener lens.APIOpener
	closer lens.APICloser
}

func NewTask(opener lens.APIOpener) *Task {
	return &Task{
		opener: opener,
	}
}

func (p *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
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

	ce, err := ExtractChainEconomicsModel(ctx, p.node, ts)
	if err != nil {
		log.Errorw("error received while extracting chain economics, closing lens", "error", err)
		if cerr := p.Close(); cerr != nil {
			log.Errorw("error received while closing lens", "error", cerr)
		}
		return nil, nil, err
	}

	return ce, report, nil
}

func (p *Task) Close() error {
	if p.closer != nil {
		p.closer()
		p.closer = nil
	}
	p.node = nil
	return nil
}

// Package msapprovals provides a task for recording multisig approvals

package msapprovals

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

var log = logging.Logger("msapprovals")

type Task struct {
	node       lens.API
	opener     lens.APIOpener
	closer     lens.APICloser
	lastTipSet *types.TipSet
}

func NewTask(opener lens.APIOpener) *Task {
	return &Task{
		opener: opener,
	}
}

func (p *Task) ProcessMessages(ctx context.Context, ts *types.TipSet, pts *types.TipSet, emsgs []*lens.ExecutedMessage) (model.Persistable, *visormodel.ProcessingReport, error) {
	// TODO: refactor this boilerplate into a helper
	if p.node == nil {
		node, closer, err := p.opener.Open(ctx)
		if err != nil {
			return nil, nil, xerrors.Errorf("unable to open lens: %w", err)
		}
		p.node = node
		p.closer = closer
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(pts.Height()),
		StateRoot: pts.ParentState().String(),
	}

	ll := log.With("height", int64(ts.Height()))

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		// ll.Infow("found message", "to", m.ToActorCode.String(), "addr", m.Message.To.String())

		if !isMultisigActor(m.ToActorCode) {
			continue
		}

		ll.Infow("found multisig", "addr", m.Message.To.String())

	}

	return nil, report, nil
}

func (p *Task) Close() error {
	if p.closer != nil {
		p.closer()
		p.closer = nil
	}
	p.node = nil
	return nil
}

func isMultisigActor(code cid.Cid) bool {
	return code == sa0builtin.MultisigActorCodeID || code == sa2builtin.MultisigActorCodeID || code == sa3builtin.MultisigActorCodeID
}

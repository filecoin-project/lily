// Package msapprovals provides a task for recording multisig approvals

package msapprovals

import (
	"bytes"
	"context"

	"github.com/filecoin-project/lotus/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lotus/chain/types"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/msapprovals"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

var log = logging.Logger("msapprovals")

const (
	ProposeMethodNum = 2
	ApproveMethodNum = 3
)

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

	ll := log.With("height", int64(pts.Height()))

	errorsDetected := make([]*MultisigError, 0, len(emsgs))
	results := make(msapprovals.MultisigApprovalList, 0) // no inital size capacity since approvals are rare

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		// ll.Infow("found message", "to", m.ToActorCode.String(), "addr", m.Message.To.String())

		// Only interested in messages to multisig actors
		if !isMultisigActor(m.ToActorCode) {
			continue
		}

		// Only interested in successful messages
		if !m.Receipt.ExitCode.IsSuccess() {
			continue
		}

		ll.Infow("found multisig", "addr", m.Message.To.String(), "method", m.Message.Method, "exit_code", m.Receipt.ExitCode, "gas_used", m.Receipt.GasUsed)

		// Only interested in propose and approve messages
		if m.Message.Method != ProposeMethodNum && m.Message.Method != ApproveMethodNum {
			continue
		}

		// The return value will tell us if the multisig was approved
		var ret multisig.ProposeReturn
		err := ret.UnmarshalCBOR(bytes.NewReader(m.Receipt.Return))
		if err != nil {
			errorsDetected = append(errorsDetected, &MultisigError{
				Addr:  m.Message.To.String(),
				Error: xerrors.Errorf("failed to decode return value: %w", err).Error(),
			})
			continue
		}

		ll.Infow("found multisig", "txn_id", ret.TxnID, "applied", ret.Applied, "code", ret.Code)

		// Get state of actor after the message has been applied
		act, err := p.node.StateGetActor(ctx, m.Message.To, ts.Key())
		if err != nil {
			errorsDetected = append(errorsDetected, &MultisigError{
				Addr:  m.Message.To.String(),
				Error: xerrors.Errorf("failed to load actor: %w", err).Error(),
			})
			continue
		}

		actorState, err := multisig.Load(p.node.Store(), act)
		if err != nil {
			errorsDetected = append(errorsDetected, &MultisigError{
				Addr:  m.Message.To.String(),
				Error: xerrors.Errorf("failed to load actor state: %w", err).Error(),
			})
			continue
		}

		appr := msapprovals.MultisigApproval{
			Height:        int64(pts.Height()),
			StateRoot:     pts.ParentState().String(),
			MultisigID:    m.Message.To.String(),
			Message:       m.Cid.String(),
			TransactionID: int64(ret.TxnID),
			Method:        uint64(m.Message.Method),
			Approver:      m.Message.From.String(),
		}

		ib, err := actorState.InitialBalance()
		if err != nil {
			errorsDetected = append(errorsDetected, &MultisigError{
				Addr:  m.Message.To.String(),
				Error: xerrors.Errorf("failed to read initial balance: %w", err).Error(),
			})
			continue
		}
		appr.InitialBalance = ib.String()

		signers, err := actorState.Signers()
		if err != nil {
			errorsDetected = append(errorsDetected, &MultisigError{
				Addr:  m.Message.To.String(),
				Error: xerrors.Errorf("failed to read signers: %w", err).Error(),
			})
			continue
		}
		for _, addr := range signers {
			appr.Signers = append(appr.Signers, addr.String())
		}

		log.Debugf("MultisigApproval: %+v", appr)

		results = append(results, &appr)

		// ExitCode exitcode.ExitCode
		// Return   []byte
		// GasUsed  int64

		// previb, _ := ec.PrevState.InitialBalance()
		// prevlb, _ := ec.PrevState.LockedBalance(ec.CurrTs.Height() - 1)
		// prevud, _ := ec.PrevState.UnlockDuration()
		// prevthres, _ := ec.PrevState.Threshold()
		// prevsigners, _ := ec.PrevState.Signers()
		// log.Debugw("multisig previous state", "initial_balance", previb, "locked_balance", prevlb, "unlock_duration", prevud, "threshold", prevthres, "signers", len(prevsigners))

		// approved := make([]string, len(added.Tx.Approved))
		// for i, addr := range added.Tx.Approved {
		// 	approved[i] = addr.String()
		// }

	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return results, report, nil
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

type MultisigError struct {
	Addr  string
	Error string
}

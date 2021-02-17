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
	multisig3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"
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
			Height:     int64(pts.Height()),
			StateRoot:  pts.ParentState().String(),
			MultisigID: m.Message.To.String(),
			Message:    m.Cid.String(),
			Method:     uint64(m.Message.Method),
			Approver:   m.Message.From.String(),
			GasUsed:    m.Receipt.GasUsed,
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

		threshold, err := actorState.Threshold()
		if err != nil {
			errorsDetected = append(errorsDetected, &MultisigError{
				Addr:  m.Message.To.String(),
				Error: xerrors.Errorf("failed to read initial balance: %w", err).Error(),
			})
			continue
		}
		appr.Threshold = threshold

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

		// If the message is a proposal then the parameters will contain details of the transaction
		if m.Message.Method == ProposeMethodNum {
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

			// this type is the same between v0 and v3
			var params multisig3.ProposeParams
			err = params.UnmarshalCBOR(bytes.NewReader(m.Message.Params))
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  m.Message.To.String(),
					Error: xerrors.Errorf("failed to decode message params: %w", err).Error(),
				})
				continue
			}

			appr.TransactionID = int64(ret.TxnID)
			appr.To = params.To.String()
			appr.Value = params.Value.String()

		} else if m.Message.Method == ApproveMethodNum {
			// If the message is an approve then the params will contain the id of a pending transaction

			// this type is the same between v0 and v3
			var params multisig3.TxnIDParams
			err = params.UnmarshalCBOR(bytes.NewReader(m.Message.Params))
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  m.Message.To.String(),
					Error: xerrors.Errorf("failed to decode message params: %w", err).Error(),
				})
				continue
			}

			appr.TransactionID = int64(params.ID)

			// Get state of actor before the message was applied
			act, err := p.node.StateGetActor(ctx, m.Message.To, pts.Key())
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  m.Message.To.String(),
					Error: xerrors.Errorf("failed to load previous actor: %w", err).Error(),
				})
				continue
			}

			prevActorState, err := multisig.Load(p.node.Store(), act)
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  m.Message.To.String(),
					Error: xerrors.Errorf("failed to load previous actor state: %w", err).Error(),
				})
				continue
			}

			if err := prevActorState.ForEachPendingTxn(func(id int64, txn multisig.Transaction) error {
				if id == int64(params.ID) {
					appr.To = txn.To.String()
					appr.Value = txn.Value.String()
					return nil
				}
				return xerrors.Errorf("pending transaction %d not found", params.ID)
			}); err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  m.Message.To.String(),
					Error: xerrors.Errorf("failed to read transaction details: %w", err).Error(),
				})
				continue

			}
		}

		log.Debugf("MultisigApproval: %+v", appr)

		results = append(results, &appr)
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

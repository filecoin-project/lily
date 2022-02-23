// Package msapprovals provides a task for recording multisig approvals
package msapprovals

import (
	"bytes"
	"context"

	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/msapprovals"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

const (
	ProposeMethodNum = 2
	ApproveMethodNum = 3
)

type Task struct {
	node lens.API
}

func NewTask(node lens.API) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessMessages(ctx context.Context, ts *types.TipSet, pts *types.TipSet, emsgs []*lens.ExecutedMessage, _ []*lens.BlockMessages) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessMultisigApprovals")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.String()), attribute.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(pts.Height()),
		StateRoot: pts.ParentState().String(),
	}

	errorsDetected := make([]*MultisigError, 0, len(emsgs))
	results := make(msapprovals.MultisigApprovalList, 0) // no initial size capacity since approvals are rare

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		// Only interested in messages to multisig actors
		if !builtin.IsMultisigActor(m.ToActorCode) {
			continue
		}

		// Only interested in successful messages
		if !m.Receipt.ExitCode.IsSuccess() {
			continue
		}

		// Only interested in propose and approve messages
		if m.Message.Method != ProposeMethodNum && m.Message.Method != ApproveMethodNum {
			continue
		}

		applied, tx, err := p.getTransactionIfApplied(ctx, m.Message, m.Receipt, ts)
		if err != nil {
			errorsDetected = append(errorsDetected, &MultisigError{
				Addr:  m.Message.To.String(),
				Error: xerrors.Errorf("failed to find transaction: %w", err).Error(),
			})
			continue
		}

		// Only interested in messages that applied a transaction
		if !applied {
			continue
		}

		appr := msapprovals.MultisigApproval{
			Height:        int64(pts.Height()),
			StateRoot:     pts.ParentState().String(),
			MultisigID:    m.Message.To.String(),
			Message:       m.Cid.String(),
			Method:        uint64(m.Message.Method),
			Approver:      m.Message.From.String(),
			GasUsed:       m.Receipt.GasUsed,
			TransactionID: tx.id,
			To:            tx.to,
			Value:         tx.value,
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

		results = append(results, &appr)
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return results, report, nil
}

type MultisigError struct {
	Addr  string
	Error string
}

type transaction struct {
	id    int64
	to    string
	value string
}

// getTransactionIfApplied returns the transaction associated with the message if the transaction was applied (i.e. had enough
// approvals). Returns true and the transaction if the transaction was applied, false otherwise.
func (p *Task) getTransactionIfApplied(ctx context.Context, msg *types.Message, rcpt *types.MessageReceipt, pts *types.TipSet) (bool, *transaction, error) {
	switch msg.Method {
	case ProposeMethodNum:
		// If the message is a proposal then the parameters will contain details of the transaction

		// The return value will tell us if the multisig transaction was applied
		var ret multisig.ProposeReturn
		err := ret.UnmarshalCBOR(bytes.NewReader(rcpt.Return))
		if err != nil {
			return false, nil, xerrors.Errorf("failed to decode return value: %w", err)
		}

		// Only interested in applied transactions
		if !ret.Applied {
			return false, nil, nil
		}

		// this type is the same between v0 and v6
		var params multisig.ProposeParams
		err = params.UnmarshalCBOR(bytes.NewReader(msg.Params))
		if err != nil {
			return false, nil, xerrors.Errorf("failed to decode message params: %w", err)
		}

		return true, &transaction{
			id:    int64(ret.TxnID),
			to:    params.To.String(),
			value: params.Value.String(),
		}, nil

	case ApproveMethodNum:
		// If the message is an approve then the params will contain the id of a pending transaction

		// this type is the same between v0 and v7
		var ret multisig.ApproveReturn
		err := ret.UnmarshalCBOR(bytes.NewReader(rcpt.Return))
		if err != nil {
			return false, nil, xerrors.Errorf("failed to decode return value: %w", err)
		}

		// Only interested in applied transactions
		if !ret.Applied {
			return false, nil, nil
		}

		// this type is the same between v0 and v6
		var params multisig.TxnIDParams
		err = params.UnmarshalCBOR(bytes.NewReader(msg.Params))
		if err != nil {
			return false, nil, xerrors.Errorf("failed to decode message params: %w", err)
		}

		// Get state of actor before the message was applied
		// pts is the tipset containing the messages, so we need the state as seen at the start of message processing
		// for that tipset
		act, err := p.node.StateGetActor(ctx, msg.To, pts.Parents())
		if err != nil {
			return false, nil, xerrors.Errorf("failed to load previous actor: %w", err)
		}

		prevActorState, err := multisig.Load(p.node.Store(), act)
		if err != nil {
			return false, nil, xerrors.Errorf("failed to load previous actor state: %w", err)
		}

		var tx *transaction

		if err := prevActorState.ForEachPendingTxn(func(id int64, txn multisig.Transaction) error {
			if id == int64(params.ID) {
				tx = &transaction{
					id:    int64(params.ID),
					to:    txn.To.String(),
					value: txn.Value.String(),
				}
			}
			return nil
		}); err != nil {
			return false, nil, xerrors.Errorf("failed to read transaction details: %w", err)
		}

		if tx == nil {
			return false, nil, xerrors.Errorf("pending transaction %d not found", params.ID)
		}

		return true, tx, nil

	default:
		// Not interested in any other methods

		return false, nil, nil

	}
}

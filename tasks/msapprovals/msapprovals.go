// Package msapprovals provides a task for recording multisig approvals
package msapprovals

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/msapprovals"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
)

const (
	ProposeMethodNum = 2
	ApproveMethodNum = 3
)

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "multisig_approvals"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	grp, _ := errgroup.WithContext(ctx)

	var getActorCodeFn func(ctx context.Context, address address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCodeFn, err = util.MakeGetActorCodeFunc(ctx, p.node.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("getting actor code lookup function: %w", err)
		}
		return nil
	})

	var blkMsgRec []*lens.BlockMessageReceipts
	grp.Go(func() error {
		var err error
		blkMsgRec, err = p.node.TipSetMessageReceipts(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages and receipts: %w", err)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		report.ErrorsDetected = err
		return nil, report, nil
	}

	errorsDetected := make([]*MultisigError, 0)
	results := make(msapprovals.MultisigApprovalList, 0) // no initial size capacity since approvals are rare

	for _, msgrec := range blkMsgRec {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		itr, err := msgrec.Iterator()
		if err != nil {
			return nil, nil, err
		}

		for itr.HasNext() {
			msg, _, rec := itr.Next()
			// Only interested in successful messages
			if !rec.ExitCode.IsSuccess() {
				continue
			}

			// Only interested in messages to multisig actors
			msgToCode, found := getActorCodeFn(ctx, msg.VMMessage().To)
			if !found {
				return nil, nil, fmt.Errorf("failed to find to actor %s height %d message %s", msg.VMMessage().To, current.Height(), msg.Cid())
			}
			if !builtin.IsMultisigActor(msgToCode) {
				continue
			}

			// Only interested in propose and approve messages
			if msg.VMMessage().Method != ProposeMethodNum && msg.VMMessage().Method != ApproveMethodNum {
				continue
			}

			applied, tx, err := p.getTransactionIfApplied(ctx, msg.VMMessage(), rec, current)
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  msg.VMMessage().To.String(),
					Error: fmt.Errorf("failed to find transaction for message %s: %w", msg.Cid(), err).Error(),
				})
				continue
			}

			// Only interested in messages that applied a transaction
			if !applied {
				continue
			}

			appr := msapprovals.MultisigApproval{
				Height:        int64(executed.Height()),
				StateRoot:     executed.ParentState().String(),
				MultisigID:    msg.VMMessage().To.String(),
				Message:       msg.Cid().String(),
				Method:        uint64(msg.VMMessage().Method),
				Approver:      msg.VMMessage().From.String(),
				GasUsed:       rec.GasUsed,
				TransactionID: tx.id,
				To:            tx.to,
				Value:         tx.value,
			}

			// Get state of actor after the message has been applied
			act, err := p.node.Actor(ctx, msg.VMMessage().To, current.Key())
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  msg.VMMessage().To.String(),
					Error: fmt.Errorf("failed to load actor: %w", err).Error(),
				})
				continue
			}

			actorState, err := multisig.Load(p.node.Store(), act)
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  msg.VMMessage().To.String(),
					Error: fmt.Errorf("failed to load actor state: %w", err).Error(),
				})
				continue
			}

			ib, err := actorState.InitialBalance()
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  msg.VMMessage().To.String(),
					Error: fmt.Errorf("failed to read initial balance: %w", err).Error(),
				})
				continue
			}
			appr.InitialBalance = ib.String()

			threshold, err := actorState.Threshold()
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  msg.VMMessage().To.String(),
					Error: fmt.Errorf("failed to read initial balance: %w", err).Error(),
				})
				continue
			}
			appr.Threshold = threshold

			signers, err := actorState.Signers()
			if err != nil {
				errorsDetected = append(errorsDetected, &MultisigError{
					Addr:  msg.VMMessage().To.String(),
					Error: fmt.Errorf("failed to read signers: %w", err).Error(),
				})
				continue
			}
			for _, addr := range signers {
				appr.Signers = append(appr.Signers, addr.String())
			}

			results = append(results, &appr)
		}
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
			return false, nil, fmt.Errorf("failed to decode return value: %w", err)
		}

		// Only interested in applied transactions
		if !ret.Applied {
			return false, nil, nil
		}

		// this type is the same between v0 and v6
		var params multisig.ProposeParams
		err = params.UnmarshalCBOR(bytes.NewReader(msg.Params))
		if err != nil {
			return false, nil, fmt.Errorf("failed to decode message params: %w", err)
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
			return false, nil, fmt.Errorf("failed to decode return value: %w", err)
		}

		// Only interested in applied transactions
		if !ret.Applied {
			return false, nil, nil
		}

		// this type is the same between v0 and v6
		var params multisig.TxnIDParams
		err = params.UnmarshalCBOR(bytes.NewReader(msg.Params))
		if err != nil {
			return false, nil, fmt.Errorf("failed to decode message params: %w", err)
		}

		// Get state of actor before the message was applied
		// pts is the tipset containing the messages, so we need the state as seen at the start of message processing
		// for that tipset
		act, err := p.node.Actor(ctx, msg.To, pts.Parents())
		if err != nil {
			return false, nil, fmt.Errorf("failed to load previous actor: %w", err)
		}

		prevActorState, err := multisig.Load(p.node.Store(), act)
		if err != nil {
			return false, nil, fmt.Errorf("failed to load previous actor state: %w", err)
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
			return false, nil, fmt.Errorf("failed to read transaction details: %w", err)
		}

		if tx == nil {
			return false, nil, fmt.Errorf("pending transaction %d not found", params.ID)
		}

		return true, tx, nil

	default:
		// Not interested in any other methods

		return false, nil, nil

	}
}

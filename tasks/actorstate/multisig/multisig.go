package multisig

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/tasks/actorstate"

	"github.com/filecoin-project/lily/model"
	multisigmodel "github.com/filecoin-project/lily/model/actors/multisig"
)

var log = logging.Logger("lily/tasks/multisig")

type MultiSigActorExtractor struct{}

func (MultiSigActorExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "MultiSigActorExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "MultiSigExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMultiSigExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	transactionModels, err := ExtractMultisigTransactions(ctx, a, ec)
	if err != nil {
		return nil, fmt.Errorf("extracting multisig actor %s with head %s transactions: %w", a.Address, a.Actor.Head, err)
	}
	return &multisigmodel.MultisigTaskResult{TransactionModel: transactionModels}, nil
}

func ExtractMultisigTransactions(ctx context.Context, a actorstate.ActorInfo, ec *MsigExtractionContext) (multisigmodel.MultisigTransactionList, error) {
	var out multisigmodel.MultisigTransactionList
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachPendingTxn(func(id int64, txn multisig.Transaction) error {
			// the ordering of this list must always be preserved as the 0th entry is the proposer.
			approved := make([]string, len(txn.Approved))
			for i, addr := range txn.Approved {
				approved[i] = addr.String()
			}
			out = append(out, &multisigmodel.MultisigTransaction{
				MultisigID:    a.Address.String(),
				StateRoot:     ec.CurrTs.ParentState().String(),
				Height:        int64(ec.CurrTs.Height()),
				TransactionID: id,
				To:            txn.To.String(),
				Value:         txn.Value.String(),
				Method:        uint64(txn.Method),
				Params:        txn.Params,
				Approved:      approved,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}

	changes, err := multisig.DiffPendingTransactions(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, fmt.Errorf("diffing pending transactions: %w", err)
	}

	for _, added := range changes.Added {
		approved := make([]string, len(added.Tx.Approved))
		for i, addr := range added.Tx.Approved {
			approved[i] = addr.String()
		}
		out = append(out, &multisigmodel.MultisigTransaction{
			MultisigID:    a.Address.String(),
			StateRoot:     a.Current.ParentState().String(),
			Height:        int64(ec.CurrTs.Height()),
			TransactionID: added.TxID,
			To:            added.Tx.To.String(),
			Value:         added.Tx.Value.String(),
			Method:        uint64(added.Tx.Method),
			Params:        added.Tx.Params,
			Approved:      approved,
		})
	}

	for _, modded := range changes.Modified {
		approved := make([]string, len(modded.To.Approved))
		for i, addr := range modded.To.Approved {
			approved[i] = addr.String()
		}
		out = append(out, &multisigmodel.MultisigTransaction{
			MultisigID:    a.Address.String(),
			StateRoot:     a.Current.ParentState().String(),
			Height:        int64(ec.CurrTs.Height()),
			TransactionID: modded.TxID,
			To:            modded.To.To.String(),
			Value:         modded.To.Value.String(),
			Method:        uint64(modded.To.Method),
			Params:        modded.To.Params,
			Approved:      approved,
		})

	}
	return out, nil
}

type MsigExtractionContext struct {
	PrevState multisig.State

	CurrActor *types.Actor
	CurrState multisig.State
	CurrTs    *types.TipSet

	Store                adt.Store
	PreviousStatePresent bool
}

func (m *MsigExtractionContext) HasPreviousState() bool {
	return m.PreviousStatePresent
}

func NewMultiSigExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*MsigExtractionContext, error) {
	curState, err := multisig.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current multisig state at head %s: %w", a.Actor.Head, err)
	}

	prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		// actor doesn't exist yet, may have just been created.
		if err == types.ErrActorNotFound {
			return &MsigExtractionContext{
				CurrActor:            &a.Actor,
				CurrState:            curState,
				CurrTs:               a.Current,
				Store:                node.Store(),
				PrevState:            nil,
				PreviousStatePresent: false,
			}, nil
		}
		return nil, fmt.Errorf("loading previous multisig %s from parent tipset %s current epoch %d: %w", a.Address, a.Executed.Key(), a.Current.Height(), err)
	}

	// actor exists in previous state, load it.
	prevState, err := multisig.Load(node.Store(), prevActor)
	if err != nil {
		return nil, fmt.Errorf("loading previous multisig actor state: %w", err)
	}

	return &MsigExtractionContext{
		PrevState:            prevState,
		CurrActor:            &a.Actor,
		CurrState:            curState,
		CurrTs:               a.Current,
		Store:                node.Store(),
		PreviousStatePresent: true,
	}, nil
}

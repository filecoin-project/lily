package actorstate

import (
	"context"

	"github.com/filecoin-project/lotus/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lotus/chain/types"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	multisigmodel "github.com/filecoin-project/sentinel-visor/model/actors/multisig"
)

func init() {
	Register(sa0builtin.MultisigActorCodeID, MultiSigActorExtractor{})
	Register(sa2builtin.MultisigActorCodeID, MultiSigActorExtractor{})
	Register(sa3builtin.MultisigActorCodeID, MultiSigActorExtractor{})
}

type MultiSigActorExtractor struct{}

func (m MultiSigActorExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "MultiSigActor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	ec, err := NewMultiSigExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	transactionModels, err := ExtractMultisigTransactions(a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting multisig actor %s with head %s transactions: %w", a.Address, a.Actor.Head, err)
	}
	return &multisigmodel.MultisigTaskResult{TransactionModel: transactionModels}, nil
}

func ExtractMultisigTransactions(a ActorInfo, ec *MsigExtractionContext) (multisigmodel.MultisigTransactionList, error) {
	var out multisigmodel.MultisigTransactionList
	if ec.IsGenesis() {
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

	changes, err := multisig.DiffPendingTransactions(ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, xerrors.Errorf("diffing pending transactions: %w", err)
	}

	for _, added := range changes.Added {
		approved := make([]string, len(added.Tx.Approved))
		for i, addr := range added.Tx.Approved {
			approved[i] = addr.String()
		}
		out = append(out, &multisigmodel.MultisigTransaction{
			MultisigID:    a.Address.String(),
			StateRoot:     a.ParentStateRoot.String(),
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
			StateRoot:     a.ParentStateRoot.String(),
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
}

func (m *MsigExtractionContext) IsGenesis() bool {
	return m.CurrTs.Height() == 0
}

func NewMultiSigExtractionContext(ctx context.Context, a ActorInfo, node ActorStateAPI) (*MsigExtractionContext, error) {
	curTipset, err := node.ChainGetTipSet(ctx, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current tipset %s: %w", a.TipSet.String(), err)
	}

	curState, err := multisig.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current multisig state at head %s: %w", a.Actor.Head, err)
	}

	prevState := curState
	if a.Epoch != 0 {
		prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet)
		if err != nil {
			return nil, xerrors.Errorf("loading previous multisig %s at tipset %s epoch %d: %w", a.Address, a.ParentTipSet, a.Epoch, err)
		}

		prevState, err = multisig.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous multisig actor state: %w", err)
		}
	}

	return &MsigExtractionContext{
		PrevState: prevState,
		CurrActor: &a.Actor,
		CurrState: curState,
		CurrTs:    curTipset,
	}, nil
}

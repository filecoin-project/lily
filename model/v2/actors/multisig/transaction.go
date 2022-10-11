package multisig

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&MultisigTransaction{}, ExtractMultisigTransaction)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range multisig.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&MultisigTransaction{}, supportedActors)

}

var log = logging.Logger("multisig")

var _ v2.LilyModel = (*MultisigTransaction)(nil)

type TransactionEvent int64

const (
	Added TransactionEvent = iota
	Modified
	Removed
)

func (t TransactionEvent) String() string {
	switch t {
	case Added:
		return "ADDED"
	case Modified:
		return "MODIFIED"
	case Removed:
		return "REMOVED"
	}
	panic(fmt.Sprintf("unhandled type %d developer error", t))
}

type MultisigTransaction struct {
	Height        abi.ChainEpoch
	StateRoot     cid.Cid
	Multisig      address.Address
	Event         TransactionEvent
	TransactionID int64
	To            address.Address
	Value         abi.TokenAmount
	Method        abi.MethodNum
	Params        []byte
	Approved      []address.Address
}

func (m *MultisigTransaction) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(MultisigTransaction{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *MultisigTransaction) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *MultisigTransaction) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *MultisigTransaction) ToStorageBlock() (block.Block, error) {
	data, err := m.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (m *MultisigTransaction) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractMultisigTransaction(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "MultisigTransaction"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "MultiSigExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMultiSigExtractionContext(ctx, a, api)
	if err != nil {
		return nil, err
	}

	var out []v2.LilyModel
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachPendingTxn(func(id int64, txn multisig.Transaction) error {
			out = append(out, &MultisigTransaction{
				Height:        a.Current.Height(),
				StateRoot:     a.Current.ParentState(),
				Multisig:      a.Address,
				Event:         Added,
				TransactionID: id,
				To:            txn.To,
				Value:         txn.Value,
				Method:        txn.Method,
				Params:        txn.Params,
				Approved:      txn.Approved,
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
		out = append(out, &MultisigTransaction{
			Height:        a.Current.Height(),
			StateRoot:     a.Current.ParentState(),
			Multisig:      a.Address,
			Event:         Added,
			TransactionID: added.TxID,
			To:            added.Tx.To,
			Value:         added.Tx.Value,
			Method:        added.Tx.Method,
			Params:        added.Tx.Params,
			Approved:      added.Tx.Approved,
		})
	}

	for _, modded := range changes.Modified {
		out = append(out, &MultisigTransaction{
			Height:        a.Current.Height(),
			StateRoot:     a.Current.ParentState(),
			Multisig:      a.Address,
			Event:         Modified,
			TransactionID: modded.TxID,
			To:            modded.To.To,
			Value:         modded.To.Value,
			Method:        modded.To.Method,
			Params:        modded.To.Params,
			Approved:      modded.To.Approved,
		})
	}

	for _, removed := range changes.Removed {
		out = append(out, &MultisigTransaction{
			Height:        a.Current.Height(),
			StateRoot:     a.Current.ParentState(),
			Multisig:      a.Address,
			Event:         Removed,
			TransactionID: removed.TxID,
			To:            removed.Tx.To,
			Value:         removed.Tx.Value,
			Method:        removed.Tx.Method,
			Params:        removed.Tx.Params,
			Approved:      removed.Tx.Approved,
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

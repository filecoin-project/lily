package multisig

import (
	"context"

	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

type PendingTransactionChanges struct {
	Added    []TransactionChange
	Modified []TransactionModification
	Removed  []TransactionChange
}

type TransactionChange struct {
	TxID int64
	Tx   Transaction
}

type TransactionModification struct {
	TxID int64
	From Transaction
	To   Transaction
}

func DiffPendingTransactions(ctx context.Context, store adt.Store, pre, cur State) (*PendingTransactionChanges, error) {
	pret, err := pre.PendingTransactionsMap()
	if err != nil {
		return nil, err
	}

	curt, err := cur.PendingTransactionsMap()
	if err != nil {
		return nil, err
	}

	diffContainer := NewTransactionDiffContainer(pre, cur)
	if requiresLegacyDiffing(pre, cur,
		&adt.MapOpts{
			Bitwidth: pre.PendingTransactionsMapBitWidth(),
			HashFunc: pre.PendingTransactionsMapHashFunction(),
		},
		&adt.MapOpts{
			Bitwidth: cur.PendingTransactionsMapBitWidth(),
			HashFunc: pre.PendingTransactionsMapHashFunction(),
		}) {
		if err := diff.CompareMap(pret, curt, diffContainer); err != nil {
			return nil, err
		}
		return diffContainer.Results, nil
	}

	changes, err := diff.Hamt(ctx, pret, curt, store, store, hamt.UseTreeBitWidth(pre.PendingTransactionsMapBitWidth()), hamt.UseHashFunction(pre.PendingTransactionsMapHashFunction()))
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		switch change.Type {
		case hamt.Add:
			if err := diffContainer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case hamt.Remove:
			if err := diffContainer.Remove(change.Key, change.Before); err != nil {
				return nil, err
			}
		case hamt.Modify:
			if err := diffContainer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		}
	}

	return diffContainer.Results, nil
}

func NewTransactionDiffContainer(pre, cur State) *transactionDiffContainer {
	return &transactionDiffContainer{
		Results: new(PendingTransactionChanges),
		pre:     pre,
		after:   cur,
	}
}

type transactionDiffContainer struct {
	Results    *PendingTransactionChanges
	pre, after State
}

func (t *transactionDiffContainer) AsKey(key string) (abi.Keyer, error) {
	txID, err := abi.ParseIntKey(key)
	if err != nil {
		return nil, err
	}
	return abi.IntKey(txID), nil
}

func (t *transactionDiffContainer) Add(key string, val *cbg.Deferred) error {
	txID, err := abi.ParseIntKey(key)
	if err != nil {
		return err
	}
	tx, err := t.after.decodeTransaction(val)
	if err != nil {
		return err
	}
	t.Results.Added = append(t.Results.Added, TransactionChange{
		TxID: txID,
		Tx:   tx,
	})
	return nil
}

func (t *transactionDiffContainer) Modify(key string, from, to *cbg.Deferred) error {
	txID, err := abi.ParseIntKey(key)
	if err != nil {
		return err
	}

	txFrom, err := t.pre.decodeTransaction(from)
	if err != nil {
		return err
	}

	txTo, err := t.after.decodeTransaction(to)
	if err != nil {
		return err
	}

	if approvalsChanged(txFrom.Approved, txTo.Approved) {
		t.Results.Modified = append(t.Results.Modified, TransactionModification{
			TxID: txID,
			From: txFrom,
			To:   txTo,
		})
	}

	return nil
}

func approvalsChanged(from, to []address.Address) bool {
	if len(from) != len(to) {
		return true
	}
	for idx := range from {
		if from[idx] != to[idx] {
			return true
		}
	}
	return false
}

func (t *transactionDiffContainer) Remove(key string, val *cbg.Deferred) error {
	txID, err := abi.ParseIntKey(key)
	if err != nil {
		return err
	}
	tx, err := t.pre.decodeTransaction(val)
	if err != nil {
		return err
	}
	t.Results.Removed = append(t.Results.Removed, TransactionChange{
		TxID: txID,
		Tx:   tx,
	})
	return nil
}

func requiresLegacyDiffing(pre, cur State, pOpts, cOpts *adt.MapOpts) bool {
	// hamt/v3 cannot read hamt/v2 nodes. Their Pointers struct has changed cbor marshalers.
	if pre.Code() == builtin0.MultisigActorCodeID {
		return true
	}
	if pre.Code() == builtin2.MultisigActorCodeID {
		return true
	}
	if cur.Code() == builtin0.MultisigActorCodeID {
		return true
	}
	if cur.Code() == builtin2.MultisigActorCodeID {
		return true
	}
	// bitwidth or hashfunction differences mean legacy diffing.
	if !pOpts.Equal(cOpts) {
		return true
	}
	return false
}

// Code generated by: `make actors-gen`. DO NOT EDIT.
package multisig

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"

	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/manifest"

	"crypto/sha256"

	builtin9 "github.com/filecoin-project/go-state-types/builtin"
	msig9 "github.com/filecoin-project/go-state-types/builtin/v9/multisig"
	adt9 "github.com/filecoin-project/go-state-types/builtin/v9/util/adt"
)

var _ State = (*state9)(nil)

func load9(store adt.Store, root cid.Cid) (State, error) {
	out := state9{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state9 struct {
	msig9.State
	store adt.Store
}

func (s *state9) LockedBalance(currEpoch abi.ChainEpoch) (abi.TokenAmount, error) {
	return s.State.AmountLocked(currEpoch - s.State.StartEpoch), nil
}

func (s *state9) StartEpoch() (abi.ChainEpoch, error) {
	return s.State.StartEpoch, nil
}

func (s *state9) UnlockDuration() (abi.ChainEpoch, error) {
	return s.State.UnlockDuration, nil
}

func (s *state9) InitialBalance() (abi.TokenAmount, error) {
	return s.State.InitialBalance, nil
}

func (s *state9) Threshold() (uint64, error) {
	return s.State.NumApprovalsThreshold, nil
}

func (s *state9) Signers() ([]address.Address, error) {
	return s.State.Signers, nil
}

func (s *state9) ForEachPendingTxn(cb func(id int64, txn Transaction) error) error {
	arr, err := adt9.AsMap(s.store, s.State.PendingTxns, builtin9.DefaultHamtBitwidth)
	if err != nil {
		return err
	}
	var out msig9.Transaction
	return arr.ForEach(&out, func(key string) error {
		txid, n := binary.Varint([]byte(key))
		if n <= 0 {
			return fmt.Errorf("invalid pending transaction key: %v", key)
		}
		return cb(txid, (Transaction)(out)) //nolint:unconvert
	})
}

func (s *state9) PendingTxnChanged(other State) (bool, error) {
	other9, ok := other.(*state9)
	if !ok {
		// treat an upgrade as a change, always
		return true, nil
	}
	return !s.State.PendingTxns.Equals(other9.PendingTxns), nil
}

func (s *state9) PendingTransactionsMap() (adt.Map, error) {
	return adt9.AsMap(s.store, s.PendingTxns, builtin9.DefaultHamtBitwidth)
}

func (s *state9) PendingTransactionsMapBitWidth() int {

	return builtin9.DefaultHamtBitwidth

}

func (s *state9) PendingTransactionsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state9) decodeTransaction(val *cbg.Deferred) (Transaction, error) {
	var tx msig9.Transaction
	if err := tx.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return Transaction{}, err
	}
	return Transaction(tx), nil
}

func (s *state9) ActorKey() string {
	return manifest.MultisigKey
}

func (s *state9) ActorVersion() actorstypes.Version {
	return actorstypes.Version9
}

func (s *state9) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}

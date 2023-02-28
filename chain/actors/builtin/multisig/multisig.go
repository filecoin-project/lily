// Code generated by: `make actors-gen`. DO NOT EDIT.
package multisig

import (
	"fmt"

	"github.com/minio/blake2b-simd"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	msig13 "github.com/filecoin-project/go-state-types/builtin/v13/multisig"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"

	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"

	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"

	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"

	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"

	builtintypes "github.com/filecoin-project/go-state-types/builtin"

	"github.com/filecoin-project/lily/chain/actors"
	"github.com/filecoin-project/lily/chain/actors/adt"
	lotusactors "github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
)

func Load(store adt.Store, act *types.Actor) (State, error) {
	if name, av, ok := lotusactors.GetActorMetaByCode(act.Code); ok {
		if name != actors.MultisigKey {
			return nil, fmt.Errorf("actor code is not multisig: %s", name)
		}

		switch actors.Version(av) {

		case actors.Version8:
			return load8(store, act.Head)

		case actors.Version9:
			return load9(store, act.Head)

		case actors.Version10:
			return load10(store, act.Head)

		case actors.Version11:
			return load11(store, act.Head)

		case actors.Version12:
			return load12(store, act.Head)

		case actors.Version13:
			return load13(store, act.Head)

		}
	}

	switch act.Code {

	case builtin0.MultisigActorCodeID:
		return load0(store, act.Head)

	case builtin2.MultisigActorCodeID:
		return load2(store, act.Head)

	case builtin3.MultisigActorCodeID:
		return load3(store, act.Head)

	case builtin4.MultisigActorCodeID:
		return load4(store, act.Head)

	case builtin5.MultisigActorCodeID:
		return load5(store, act.Head)

	case builtin6.MultisigActorCodeID:
		return load6(store, act.Head)

	case builtin7.MultisigActorCodeID:
		return load7(store, act.Head)

	}

	return nil, fmt.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	Code() cid.Cid
	ActorKey() string
	ActorVersion() actors.Version

	LockedBalance(epoch abi.ChainEpoch) (abi.TokenAmount, error)
	StartEpoch() (abi.ChainEpoch, error)
	UnlockDuration() (abi.ChainEpoch, error)
	InitialBalance() (abi.TokenAmount, error)
	Threshold() (uint64, error)
	Signers() ([]address.Address, error)

	ForEachPendingTxn(func(id int64, txn Transaction) error) error
	PendingTxnChanged(State) (bool, error)

	PendingTransactionsMap() (adt.Map, error)
	PendingTransactionsMapBitWidth() int
	PendingTransactionsMapHashFunction() func(input []byte) []byte
	decodeTransaction(val *cbg.Deferred) (Transaction, error)
}

type Transaction = msig13.Transaction

var Methods = builtintypes.MethodsMultisig

// these types are the same between v0 and v6
type ProposalHashData = msig13.ProposalHashData
type ProposeReturn = msig13.ProposeReturn
type ProposeParams = msig13.ProposeParams
type ApproveReturn = msig13.ApproveReturn
type TxnIDParams = msig13.TxnIDParams

func txnParams(id uint64, data *ProposalHashData) ([]byte, error) {
	params := msig13.TxnIDParams{ID: msig13.TxnID(id)}
	if data != nil {
		if data.Requester.Protocol() != address.ID {
			return nil, fmt.Errorf("proposer address must be an ID address, was %s", data.Requester)
		}
		if data.Value.Sign() == -1 {
			return nil, fmt.Errorf("proposal value must be non-negative, was %s", data.Value)
		}
		if data.To == address.Undef {
			return nil, fmt.Errorf("proposed destination address must be set")
		}
		pser, err := data.Serialize()
		if err != nil {
			return nil, err
		}
		hash := blake2b.Sum256(pser)
		params.ProposalHash = hash[:]
	}

	return actors.SerializeParams(&params)
}

func AllCodes() []cid.Cid {
	return []cid.Cid{
		(&state0{}).Code(),
		(&state2{}).Code(),
		(&state3{}).Code(),
		(&state4{}).Code(),
		(&state5{}).Code(),
		(&state6{}).Code(),
		(&state7{}).Code(),
		(&state8{}).Code(),
		(&state9{}).Code(),
		(&state10{}).Code(),
		(&state11{}).Code(),
		(&state12{}).Code(),
		(&state13{}).Code(),
	}
}

func VersionCodes() map[actors.Version]cid.Cid {
	return map[actors.Version]cid.Cid{
		actors.Version0:  (&state0{}).Code(),
		actors.Version2:  (&state2{}).Code(),
		actors.Version3:  (&state3{}).Code(),
		actors.Version4:  (&state4{}).Code(),
		actors.Version5:  (&state5{}).Code(),
		actors.Version6:  (&state6{}).Code(),
		actors.Version7:  (&state7{}).Code(),
		actors.Version8:  (&state8{}).Code(),
		actors.Version9:  (&state9{}).Code(),
		actors.Version10: (&state10{}).Code(),
		actors.Version11: (&state11{}).Code(),
		actors.Version12: (&state12{}).Code(),
		actors.Version13: (&state13{}).Code(),
	}
}

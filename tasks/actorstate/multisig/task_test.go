package multisig_test

// TODO break up tests per task
/*
import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/actor"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/multisig"
	actortesting "github.com/filecoin-project/sentinel-visor/tasks/actorstate/testing"
	"github.com/filecoin-project/specs-actors/actors/runtime"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	multisig0 "github.com/filecoin-project/specs-actors/actors/builtin/multisig"
	adt0 "github.com/filecoin-project/specs-actors/actors/util/adt"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	multisig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	multisig3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	sa4builtin "github.com/filecoin-project/specs-actors/v4/actors/builtin"

	multisigmodel "github.com/filecoin-project/sentinel-visor/model/actors/multisig"
)


func TestMultisigExtractorV0(t *testing.T) {
	ctx := context.Background()

	mapi := actortesting.NewMockAPI(t)
	minerAddr := tutils.NewIDAddr(t, 1234)

	emptyPending, err := adt0.MakeEmptyMap(mapi.Store()).Root()
	require.NoError(t, err)

	multiSigAddress := tutils.NewIDAddr(t, 9999)
	emptyTxState := &multisig0.State{
		Signers:               []address.Address{tutils.NewIDAddr(t, 1234)},
		NumApprovalsThreshold: 1,
		NextTxnID:             0,
		InitialBalance:        abi.NewTokenAmount(0),
		StartEpoch:            0,
		UnlockDuration:        0,
		PendingTxns:           emptyPending,
	}

	t.Run("single transaction added", func(t *testing.T) {
		// initialize with empty transaction state
		emptyTxStateCid, err := mapi.Store().Put(ctx, emptyTxState)
		require.NoError(t, err)

		emptyTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(emptyTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: emptyTxStateCid})

		//
		// add a transaction in subsequent state.
		pendingMap, err := adt0.AsMap(mapi.Store(), emptyTxState.PendingTxns)
		require.NoError(t, err)

		expectedTx := &multisig0.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa0builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		expectedTxID := multisig0.TxnID(1)
		require.NoError(t, pendingMap.Put(expectedTxID, expectedTx))

		// copy empty state and modify
		newTxState := *emptyTxState
		newTxState.PendingTxns, err = pendingMap.Root()
		require.NoError(t, err)

		txStateCid, err := mapi.Store().Put(ctx, &newTxState)
		require.NoError(t, err)

		txStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(txStateTs.Key(), multiSigAddress, &types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: txStateCid})

		//
		// create actor info, previous state has no transaction, current state has a single transaction
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: txStateCid},
			Epoch:        1, // not genesis
			Address:      multiSigAddress,
			TipSet:       txStateTs,
			ParentTipSet: emptyTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		actualTx := ms.TransactionModel[0]
		assert.EqualValues(t, expectedTx.To.String(), actualTx.To)
		assert.EqualValues(t, expectedTx.Params, actualTx.Params)
		assert.EqualValues(t, expectedTx.Method, actualTx.Method)
		assert.EqualValues(t, expectedTx.Value.String(), actualTx.Value)
		assert.Len(t, actualTx.Approved, 1)
		assert.EqualValues(t, expectedTx.Approved[0].String(), actualTx.Approved[0])
	})

	t.Run("single transaction added and single transaction modified", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt0.AsMap(mapi.Store(), singleTxState.PendingTxns)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig0.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa0builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig0.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)

		singleTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(singleTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: singleTxStateCid})

		// create second tx
		txMap, err = adt0.AsMap(mapi.Store(), singleTxState.PendingTxns)
		require.NoError(t, err)
		secondTx := &multisig0.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa0builtin.MethodsAccount.PubkeyAddress,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		secondTxId := multisig0.TxnID(2)
		require.NoError(t, txMap.Put(secondTxId, secondTx))

		// modify first tx
		var firstTxMod multisig0.Transaction
		found, err := txMap.Get(firstTxID, &firstTxMod)
		require.NoError(t, err)
		require.True(t, found)
		firstTxMod.Approved = append(firstTxMod.Approved, tutils.NewIDAddr(t, 898989))
		require.NoError(t, txMap.Put(firstTxID, &firstTxMod))

		// create second state with a newly added tx and a modified tx.
		secondTxState := singleTxState
		secondTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		secondTxStateCid, err := mapi.Store().Put(ctx, &secondTxState)
		require.NoError(t, err)
		secondTxStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(secondTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: secondTxStateCid})

		//
		// create actor info, previous state has single tx, current state has a new tx and modified tx.
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: secondTxStateCid},
			Epoch:        2, // not genesis
			Address:      multiSigAddress,
			TipSet:       secondTxStateTs,
			ParentTipSet: singleTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 2)
		newTx := ms.TransactionModel[0]
		assert.EqualValues(t, secondTx.To.String(), newTx.To)
		assert.EqualValues(t, secondTx.Params, newTx.Params)
		assert.EqualValues(t, secondTx.Method, newTx.Method)
		assert.EqualValues(t, secondTx.Value.String(), newTx.Value)
		assert.Len(t, secondTx.Approved, 1)
		assert.EqualValues(t, secondTx.Approved[0].String(), newTx.Approved[0])

		modTx := ms.TransactionModel[1]
		assert.EqualValues(t, firstTxMod.To.String(), modTx.To)
		assert.EqualValues(t, firstTxMod.Params, modTx.Params)
		assert.EqualValues(t, firstTxMod.Method, modTx.Method)
		assert.EqualValues(t, firstTxMod.Value.String(), modTx.Value)
		assert.Len(t, firstTxMod.Approved, 2)
		assert.EqualValues(t, firstTxMod.Approved[0].String(), modTx.Approved[0])
		assert.EqualValues(t, firstTxMod.Approved[1].String(), modTx.Approved[1])
	})

	t.Run("genesis special case", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt0.AsMap(mapi.Store(), singleTxState.PendingTxns)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig0.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa0builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig0.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)
		genesisTs := mapi.FakeTipset(minerAddr, 1, actortesting.WithHeight(1))
		mapi.SetActor(genesisTs.Key(), multiSigAddress, &types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: singleTxStateCid})

		info := actor.ActorInfo{
			Actor:   types.Actor{Code: sa0builtin.MultisigActorCodeID, Head: singleTxStateCid},
			Epoch:   1, // genesis
			Address: multiSigAddress,
			TipSet:  genesisTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		singleTx := ms.TransactionModel[0]
		assert.EqualValues(t, firstTx.To.String(), singleTx.To)
		assert.EqualValues(t, firstTx.Params, singleTx.Params)
		assert.EqualValues(t, firstTx.Method, singleTx.Method)
		assert.EqualValues(t, firstTx.Value.String(), singleTx.Value)
		assert.Len(t, firstTx.Approved, 1)
		assert.EqualValues(t, firstTx.Approved[0].String(), singleTx.Approved[0])
	})
}

func TestMultisigExtractorV2(t *testing.T) {
	ctx := context.Background()

	mapi := actortesting.NewMockAPI(t)
	minerAddr := tutils.NewIDAddr(t, 1234)

	emptyPending, err := adt2.MakeEmptyMap(mapi.Store()).Root()
	require.NoError(t, err)

	multiSigAddress := tutils.NewIDAddr(t, 9999)
	emptyTxState := &multisig2.State{
		Signers:               []address.Address{tutils.NewIDAddr(t, 1234)},
		NumApprovalsThreshold: 1,
		NextTxnID:             0,
		InitialBalance:        abi.NewTokenAmount(0),
		StartEpoch:            0,
		UnlockDuration:        0,
		PendingTxns:           emptyPending,
	}

	t.Run("single transaction added", func(t *testing.T) {
		// initialize with empty transaction state
		emptyTxStateCid, err := mapi.Store().Put(ctx, emptyTxState)
		require.NoError(t, err)

		emptyTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(emptyTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: emptyTxStateCid})

		//
		// add a transaction in subsequent state.
		pendingMap, err := adt2.AsMap(mapi.Store(), emptyTxState.PendingTxns)
		require.NoError(t, err)

		expectedTx := &multisig2.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa2builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		expectedTxID := multisig2.TxnID(1)
		require.NoError(t, pendingMap.Put(expectedTxID, expectedTx))

		// copy empty state and modify
		newTxState := *emptyTxState
		newTxState.PendingTxns, err = pendingMap.Root()
		require.NoError(t, err)

		txStateCid, err := mapi.Store().Put(ctx, &newTxState)
		require.NoError(t, err)

		txStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(txStateTs.Key(), multiSigAddress, &types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: txStateCid})

		//
		// create actor info, previous state has no transaction, current state has a single transaction
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: txStateCid},
			Epoch:        1, // not genesis
			Address:      multiSigAddress,
			TipSet:       txStateTs,
			ParentTipSet: emptyTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		actualTx := ms.TransactionModel[0]
		assert.EqualValues(t, expectedTx.To.String(), actualTx.To)
		assert.EqualValues(t, expectedTx.Params, actualTx.Params)
		assert.EqualValues(t, expectedTx.Method, actualTx.Method)
		assert.EqualValues(t, expectedTx.Value.String(), actualTx.Value)
		assert.Len(t, actualTx.Approved, 1)
		assert.EqualValues(t, expectedTx.Approved[0].String(), actualTx.Approved[0])
	})

	t.Run("single transaction added and single transaction modified", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt2.AsMap(mapi.Store(), singleTxState.PendingTxns)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig2.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa2builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig2.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)
		singleTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(singleTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: singleTxStateCid})

		// create second tx
		txMap, err = adt2.AsMap(mapi.Store(), singleTxState.PendingTxns)
		require.NoError(t, err)
		secondTx := &multisig2.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa2builtin.MethodsAccount.PubkeyAddress,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		secondTxId := multisig2.TxnID(2)
		require.NoError(t, txMap.Put(secondTxId, secondTx))

		// modify first tx
		var firstTxMod multisig2.Transaction
		found, err := txMap.Get(firstTxID, &firstTxMod)
		require.NoError(t, err)
		require.True(t, found)
		firstTxMod.Approved = append(firstTxMod.Approved, tutils.NewIDAddr(t, 898989))
		require.NoError(t, txMap.Put(firstTxID, &firstTxMod))

		// create second state with a newly added tx and a modified tx.
		secondTxState := singleTxState
		secondTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		secondTxStateCid, err := mapi.Store().Put(ctx, &secondTxState)
		require.NoError(t, err)

		secondTxStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(secondTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: secondTxStateCid})

		//
		// create actor info, previous state has single tx, current state has a new tx and modified tx.
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: secondTxStateCid},
			Epoch:        2, // not genesis
			Address:      multiSigAddress,
			TipSet:       secondTxStateTs,
			ParentTipSet: singleTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 2)
		newTx := ms.TransactionModel[0]
		assert.EqualValues(t, secondTx.To.String(), newTx.To)
		assert.EqualValues(t, secondTx.Params, newTx.Params)
		assert.EqualValues(t, secondTx.Method, newTx.Method)
		assert.EqualValues(t, secondTx.Value.String(), newTx.Value)
		assert.Len(t, secondTx.Approved, 1)
		assert.EqualValues(t, secondTx.Approved[0].String(), newTx.Approved[0])

		modTx := ms.TransactionModel[1]
		assert.EqualValues(t, firstTxMod.To.String(), modTx.To)
		assert.EqualValues(t, firstTxMod.Params, modTx.Params)
		assert.EqualValues(t, firstTxMod.Method, modTx.Method)
		assert.EqualValues(t, firstTxMod.Value.String(), modTx.Value)
		assert.Len(t, firstTxMod.Approved, 2)
		assert.EqualValues(t, firstTxMod.Approved[0].String(), modTx.Approved[0])
		assert.EqualValues(t, firstTxMod.Approved[1].String(), modTx.Approved[1])
	})

	t.Run("genesis special case", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt2.AsMap(mapi.Store(), singleTxState.PendingTxns)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig2.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa2builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig2.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)
		genesisTs := mapi.FakeTipset(minerAddr, 1, actortesting.WithHeight(0))
		mapi.SetActor(genesisTs.Key(), multiSigAddress, &types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: singleTxStateCid})

		info := actor.ActorInfo{
			Actor:   types.Actor{Code: sa2builtin.MultisigActorCodeID, Head: singleTxStateCid},
			Epoch:   1, // genesis
			Address: multiSigAddress,
			TipSet:  genesisTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		singleTx := ms.TransactionModel[0]
		assert.EqualValues(t, firstTx.To.String(), singleTx.To)
		assert.EqualValues(t, firstTx.Params, singleTx.Params)
		assert.EqualValues(t, firstTx.Method, singleTx.Method)
		assert.EqualValues(t, firstTx.Value.String(), singleTx.Value)
		assert.Len(t, firstTx.Approved, 1)
		assert.EqualValues(t, firstTx.Approved[0].String(), singleTx.Approved[0])
	})
}

func TestMultisigExtractorV3(t *testing.T) {
	ctx := context.Background()

	mapi := actortesting.NewMockAPI(t)
	minerAddr := tutils.NewIDAddr(t, 1234)

	emptyMap, err := adt3.MakeEmptyMap(mapi.Store(), sa3builtin.DefaultHamtBitwidth)
	require.NoError(t, err)

	emptyPending, err := emptyMap.Root()
	require.NoError(t, err)

	multiSigAddress := tutils.NewIDAddr(t, 9999)
	emptyTxState := &multisig3.State{
		Signers:               []address.Address{tutils.NewIDAddr(t, 1234)},
		NumApprovalsThreshold: 1,
		NextTxnID:             0,
		InitialBalance:        abi.NewTokenAmount(0),
		StartEpoch:            0,
		UnlockDuration:        0,
		PendingTxns:           emptyPending,
	}

	t.Run("single transaction added", func(t *testing.T) {
		// initialize with empty transaction state
		emptyTxStateCid, err := mapi.Store().Put(ctx, emptyTxState)
		require.NoError(t, err)

		emptyTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(emptyTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: emptyTxStateCid})

		//
		// add a transaction in subsequent state.
		pendingMap, err := adt3.AsMap(mapi.Store(), emptyTxState.PendingTxns, sa3builtin.DefaultHamtBitwidth)
		require.NoError(t, err)

		expectedTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		expectedTxID := multisig3.TxnID(1)
		require.NoError(t, pendingMap.Put(expectedTxID, expectedTx))

		// copy empty state and modify
		newTxState := *emptyTxState
		newTxState.PendingTxns, err = pendingMap.Root()
		require.NoError(t, err)

		txStateCid, err := mapi.Store().Put(ctx, &newTxState)
		require.NoError(t, err)

		txStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(txStateTs.Key(), multiSigAddress, &types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: txStateCid})

		//
		// create actor info, previous state has no transaction, current state has a single transaction
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: txStateCid},
			Epoch:        2, // not genesis
			Address:      multiSigAddress,
			TipSet:       txStateTs,
			ParentTipSet: emptyTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		actualTx := ms.TransactionModel[0]
		assert.EqualValues(t, expectedTx.To.String(), actualTx.To)
		assert.EqualValues(t, expectedTx.Params, actualTx.Params)
		assert.EqualValues(t, expectedTx.Method, actualTx.Method)
		assert.EqualValues(t, expectedTx.Value.String(), actualTx.Value)
		assert.Len(t, actualTx.Approved, 1)
		assert.EqualValues(t, expectedTx.Approved[0].String(), actualTx.Approved[0])
	})

	t.Run("single transaction added and single transaction modified", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt3.AsMap(mapi.Store(), singleTxState.PendingTxns, sa3builtin.DefaultHamtBitwidth)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig3.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)
		singleTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(singleTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: singleTxStateCid})

		// create second tx
		txMap, err = adt3.AsMap(mapi.Store(), singleTxState.PendingTxns, sa3builtin.DefaultHamtBitwidth)
		require.NoError(t, err)
		secondTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodsAccount.PubkeyAddress,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		secondTxId := multisig2.TxnID(2)
		require.NoError(t, txMap.Put(secondTxId, secondTx))

		// modify first tx
		var firstTxMod multisig2.Transaction
		found, err := txMap.Get(firstTxID, &firstTxMod)
		require.NoError(t, err)
		require.True(t, found)
		firstTxMod.Approved = append(firstTxMod.Approved, tutils.NewIDAddr(t, 898989))
		require.NoError(t, txMap.Put(firstTxID, &firstTxMod))

		// create second state with a newly added tx and a modified tx.
		secondTxState := singleTxState
		secondTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		secondTxStateCid, err := mapi.Store().Put(ctx, &secondTxState)
		require.NoError(t, err)

		secondTxStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(secondTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: secondTxStateCid})

		//
		// create actor info, previous state has single tx, current state has a new tx and modified tx.
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: secondTxStateCid},
			Epoch:        2, // not genesis
			Address:      multiSigAddress,
			TipSet:       secondTxStateTs,
			ParentTipSet: singleTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 2)
		newTx := ms.TransactionModel[0]
		assert.EqualValues(t, secondTx.To.String(), newTx.To)
		assert.EqualValues(t, secondTx.Params, newTx.Params)
		assert.EqualValues(t, secondTx.Method, newTx.Method)
		assert.EqualValues(t, secondTx.Value.String(), newTx.Value)
		assert.Len(t, secondTx.Approved, 1)
		assert.EqualValues(t, secondTx.Approved[0].String(), newTx.Approved[0])

		modTx := ms.TransactionModel[1]
		assert.EqualValues(t, firstTxMod.To.String(), modTx.To)
		assert.EqualValues(t, firstTxMod.Params, modTx.Params)
		assert.EqualValues(t, firstTxMod.Method, modTx.Method)
		assert.EqualValues(t, firstTxMod.Value.String(), modTx.Value)
		assert.Len(t, firstTxMod.Approved, 2)
		assert.EqualValues(t, firstTxMod.Approved[0].String(), modTx.Approved[0])
		assert.EqualValues(t, firstTxMod.Approved[1].String(), modTx.Approved[1])
	})

	t.Run("genesis special case", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt3.AsMap(mapi.Store(), singleTxState.PendingTxns, sa3builtin.DefaultHamtBitwidth)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig2.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)
		genesisTs := mapi.FakeTipset(minerAddr, 1, actortesting.WithHeight(0))
		mapi.SetActor(genesisTs.Key(), multiSigAddress, &types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: singleTxStateCid})

		info := actor.ActorInfo{
			Actor:   types.Actor{Code: sa3builtin.MultisigActorCodeID, Head: singleTxStateCid},
			Epoch:   1, // genesis
			Address: multiSigAddress,
			TipSet:  genesisTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		singleTx := ms.TransactionModel[0]
		assert.EqualValues(t, firstTx.To.String(), singleTx.To)
		assert.EqualValues(t, firstTx.Params, singleTx.Params)
		assert.EqualValues(t, firstTx.Method, singleTx.Method)
		assert.EqualValues(t, firstTx.Value.String(), singleTx.Value)
		assert.Len(t, firstTx.Approved, 1)
		assert.EqualValues(t, firstTx.Approved[0].String(), singleTx.Approved[0])
	})
}

func TestMultisigExtractorV4(t *testing.T) {
	ctx := context.Background()

	mapi := actortesting.NewMockAPI(t)
	minerAddr := tutils.NewIDAddr(t, 1234)

	emptyMap, err := adt3.MakeEmptyMap(mapi.Store(), sa4builtin.DefaultHamtBitwidth)
	require.NoError(t, err)

	emptyPending, err := emptyMap.Root()
	require.NoError(t, err)

	multiSigAddress := tutils.NewIDAddr(t, 9999)
	emptyTxState := &multisig3.State{
		Signers:               []address.Address{tutils.NewIDAddr(t, 1234)},
		NumApprovalsThreshold: 1,
		NextTxnID:             0,
		InitialBalance:        abi.NewTokenAmount(0),
		StartEpoch:            0,
		UnlockDuration:        0,
		PendingTxns:           emptyPending,
	}

	t.Run("single transaction added", func(t *testing.T) {
		// initialize with empty transaction state
		emptyTxStateCid, err := mapi.Store().Put(ctx, emptyTxState)
		require.NoError(t, err)

		emptyTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(emptyTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: emptyTxStateCid})

		//
		// add a transaction in subsequent state.
		pendingMap, err := adt3.AsMap(mapi.Store(), emptyTxState.PendingTxns, sa4builtin.DefaultHamtBitwidth)
		require.NoError(t, err)

		expectedTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		expectedTxID := multisig3.TxnID(1)
		require.NoError(t, pendingMap.Put(expectedTxID, expectedTx))

		// copy empty state and modify
		newTxState := *emptyTxState
		newTxState.PendingTxns, err = pendingMap.Root()
		require.NoError(t, err)

		txStateCid, err := mapi.Store().Put(ctx, &newTxState)
		require.NoError(t, err)

		txStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(txStateTs.Key(), multiSigAddress, &types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: txStateCid})

		//
		// create actor info, previous state has no transaction, current state has a single transaction
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: txStateCid},
			Epoch:        2, // not genesis
			Address:      multiSigAddress,
			TipSet:       txStateTs,
			ParentTipSet: emptyTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		actualTx := ms.TransactionModel[0]
		assert.EqualValues(t, expectedTx.To.String(), actualTx.To)
		assert.EqualValues(t, expectedTx.Params, actualTx.Params)
		assert.EqualValues(t, expectedTx.Method, actualTx.Method)
		assert.EqualValues(t, expectedTx.Value.String(), actualTx.Value)
		assert.Len(t, actualTx.Approved, 1)
		assert.EqualValues(t, expectedTx.Approved[0].String(), actualTx.Approved[0])
	})

	t.Run("single transaction added and single transaction modified", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt3.AsMap(mapi.Store(), singleTxState.PendingTxns, sa4builtin.DefaultHamtBitwidth)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig3.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)
		singleTxStateTs := mapi.FakeTipset(minerAddr, 1)
		mapi.SetActor(singleTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: singleTxStateCid})

		// create second tx
		txMap, err = adt3.AsMap(mapi.Store(), singleTxState.PendingTxns, sa3builtin.DefaultHamtBitwidth)
		require.NoError(t, err)
		secondTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodsAccount.PubkeyAddress,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		secondTxId := multisig2.TxnID(2)
		require.NoError(t, txMap.Put(secondTxId, secondTx))

		// modify first tx
		var firstTxMod multisig2.Transaction
		found, err := txMap.Get(firstTxID, &firstTxMod)
		require.NoError(t, err)
		require.True(t, found)
		firstTxMod.Approved = append(firstTxMod.Approved, tutils.NewIDAddr(t, 898989))
		require.NoError(t, txMap.Put(firstTxID, &firstTxMod))

		// create second state with a newly added tx and a modified tx.
		secondTxState := singleTxState
		secondTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		secondTxStateCid, err := mapi.Store().Put(ctx, &secondTxState)
		require.NoError(t, err)

		secondTxStateTs := mapi.FakeTipset(minerAddr, 2)
		mapi.SetActor(secondTxStateTs.Key(), multiSigAddress, &types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: secondTxStateCid})

		//
		// create actor info, previous state has single tx, current state has a new tx and modified tx.
		info := actor.ActorInfo{
			Actor:        types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: secondTxStateCid},
			Epoch:        2, // not genesis
			Address:      multiSigAddress,
			TipSet:       secondTxStateTs,
			ParentTipSet: singleTxStateTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 2)
		newTx := ms.TransactionModel[0]
		assert.EqualValues(t, secondTx.To.String(), newTx.To)
		assert.EqualValues(t, secondTx.Params, newTx.Params)
		assert.EqualValues(t, secondTx.Method, newTx.Method)
		assert.EqualValues(t, secondTx.Value.String(), newTx.Value)
		assert.Len(t, secondTx.Approved, 1)
		assert.EqualValues(t, secondTx.Approved[0].String(), newTx.Approved[0])

		modTx := ms.TransactionModel[1]
		assert.EqualValues(t, firstTxMod.To.String(), modTx.To)
		assert.EqualValues(t, firstTxMod.Params, modTx.Params)
		assert.EqualValues(t, firstTxMod.Method, modTx.Method)
		assert.EqualValues(t, firstTxMod.Value.String(), modTx.Value)
		assert.Len(t, firstTxMod.Approved, 2)
		assert.EqualValues(t, firstTxMod.Approved[0].String(), modTx.Approved[0])
		assert.EqualValues(t, firstTxMod.Approved[1].String(), modTx.Approved[1])
	})

	t.Run("genesis special case", func(t *testing.T) {
		// initialize with single transaction in state.
		singleTxState := *emptyTxState
		txMap, err := adt3.AsMap(mapi.Store(), singleTxState.PendingTxns, sa4builtin.DefaultHamtBitwidth)
		require.NoError(t, err)

		// save the new tx
		firstTx := &multisig3.Transaction{
			To:       tutils.NewIDAddr(t, 8888),
			Value:    abi.NewTokenAmount(10),
			Method:   sa3builtin.MethodSend,
			Params:   runtime.CBORBytes([]byte{1, 2, 3, 4}),
			Approved: []address.Address{tutils.NewIDAddr(t, 7777)},
		}
		firstTxID := multisig2.TxnID(1)
		require.NoError(t, txMap.Put(firstTxID, firstTx))
		singleTxState.PendingTxns, err = txMap.Root()
		require.NoError(t, err)

		// update global state
		singleTxStateCid, err := mapi.Store().Put(ctx, &singleTxState)
		require.NoError(t, err)
		genesisTs := mapi.FakeTipset(minerAddr, 1, actortesting.WithHeight(0))
		mapi.SetActor(genesisTs.Key(), multiSigAddress, &types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: singleTxStateCid})

		info := actor.ActorInfo{
			Actor:   types.Actor{Code: sa4builtin.MultisigActorCodeID, Head: singleTxStateCid},
			Epoch:   1, // genesis
			Address: multiSigAddress,
			TipSet:  genesisTs,
		}

		ex := multisig.MultiSigActorExtractor{}
		res, err := ex.Extract(ctx, info, mapi)
		require.NoError(t, err)

		ms, ok := res.(*multisigmodel.MultisigTaskResult)
		require.True(t, ok)
		require.NotNil(t, ms)

		assert.Len(t, ms.TransactionModel, 1)
		singleTx := ms.TransactionModel[0]
		assert.EqualValues(t, firstTx.To.String(), singleTx.To)
		assert.EqualValues(t, firstTx.Params, singleTx.Params)
		assert.EqualValues(t, firstTx.Method, singleTx.Method)
		assert.EqualValues(t, firstTx.Value.String(), singleTx.Value)
		assert.Len(t, firstTx.Approved, 1)
		assert.EqualValues(t, firstTx.Approved[0].String(), singleTx.Approved[0])
	})
}
*/

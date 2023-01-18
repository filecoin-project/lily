package messages

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	adtstore "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/processor"
)

type FullBlockIPLDContainer struct {
	BlockHeader  *types.BlockHeader `cborgen:"block_header"`
	SecpMessages cid.Cid            `cborgen:"secp_messages"`
	BlsMessages  cid.Cid            `cborgen:"bls_messages"`
}

func MakeFullBlockHAMT(ctx context.Context, store adtstore.Store, fullBlks map[cid.Cid]*processor.FullBlock) (cid.Cid, error) {
	fullBlkHamt, err := adt2.MakeEmptyMap(store, 5)
	if err != nil {
		return cid.Undef, err
	}

	for blkCid, fb := range fullBlks {
		blsMsgHamt, err := MakeChainMessagesHAMT(ctx, store, fb.BlsMessages)
		if err != nil {
			return cid.Undef, err
		}

		secpMsgHamt, err := MakeSignedChainMessagesHAMT(ctx, store, fb.SecpMessages)
		if err != nil {
			return cid.Undef, err
		}

		if err := fullBlkHamt.Put(abi.CidKey(blkCid), &FullBlockIPLDContainer{
			BlockHeader:  fb.Block,
			SecpMessages: secpMsgHamt,
			BlsMessages:  blsMsgHamt,
		}); err != nil {
			return cid.Undef, err
		}
	}

	return fullBlkHamt.Root()
}

type ChainMessageIPLDContainer struct {
	Message       *types.Message                 `cborgen:"message"`
	Receipt       *processor.ChainMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                        `cborgen:"vm_messages"`
}

func MakeChainMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*processor.ChainMessage) (cid.Cid, error) {
	messageHamt, err := adt2.MakeEmptyMap(store, 5)
	if err != nil {
		return cid.Undef, err
	}

	for _, msg := range messages {
		vmMsgRoot, err := msg.VmMessages.ToAdtArray(store, 5)
		if err != nil {
			return cid.Undef, err
		}
		if err := messageHamt.Put(abi.CidKey(msg.Message.Cid()), &ChainMessageIPLDContainer{
			Message:       msg.Message,
			Receipt:       msg.Receipt,
			VmMessagesAmt: vmMsgRoot,
		}); err != nil {
			return cid.Undef, err
		}
	}
	return messageHamt.Root()
}

type SignedChainMessageIPLDContainer struct {
	Message       *types.SignedMessage           `cborgen:"message"`
	Receipt       *processor.ChainMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                        `cborgen:"vm_messages"`
}

func MakeSignedChainMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*processor.SignedChainMessage) (cid.Cid, error) {
	messageHamt, err := adt2.MakeEmptyMap(store, 5)
	if err != nil {
		return cid.Undef, err
	}

	for _, msg := range messages {
		vmMsgRoot, err := msg.VmMessages.ToAdtArray(store, 5)
		if err != nil {
			return cid.Undef, err
		}
		if err := messageHamt.Put(abi.CidKey(msg.Message.Cid()), &SignedChainMessageIPLDContainer{
			Message:       msg.Message,
			Receipt:       msg.Receipt,
			VmMessagesAmt: vmMsgRoot,
		}); err != nil {
			return cid.Undef, err
		}
	}
	return messageHamt.Root()
}

type ImplicitMessageIPLDContainer struct {
	Message       *types.Message                    `cborgen:"message"`
	Receipt       *processor.ImplicitMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                           `cborgen:"vm_messages"`
}

// MakeImplicitMessagesAMT returns the root of a hamt node containing the set of implicit messages
func MakeImplicitMessagesAMT(ctx context.Context, store adtstore.Store, messages []*processor.ImplicitMessage) (cid.Cid, error) {
	messageHamt, err := adt2.MakeEmptyMap(store, 5)
	if err != nil {
		return cid.Undef, err
	}

	for _, msg := range messages {
		vmMsgRoot, err := msg.VmMessages.ToAdtArray(store, 5)
		if err != nil {
			return cid.Undef, err
		}
		if err := messageHamt.Put(abi.CidKey(msg.Message.Cid()), &ImplicitMessageIPLDContainer{
			Message:       msg.Message,
			Receipt:       msg.Receipt,
			VmMessagesAmt: vmMsgRoot,
		}); err != nil {
			return cid.Undef, err
		}
	}
	return messageHamt.Root()
}

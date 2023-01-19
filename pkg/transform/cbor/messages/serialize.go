package messages

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	adtstore "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/processor"
)

type FullBlockIPLDContainer struct {
	BlockHeader  *types.BlockHeader `cborgen:"block_header"`
	SecpMessages cid.Cid            `cborgen:"secp_messages"`
	BlsMessages  cid.Cid            `cborgen:"bls_messages"`
}

func MakeFullBlockHAMT(ctx context.Context, store adtstore.Store, fullBlks map[cid.Cid]*processor.FullBlock) (cid.Cid, error) {
	fullBlkHamt, err := adt.MakeEmptyMap(store, 5)
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

func DecodeFullBlockHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) (map[cid.Cid]*processor.FullBlock, error) {
	fullBlkHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}
	out := make(map[cid.Cid]*processor.FullBlock)
	fbc := new(FullBlockIPLDContainer)
	if err := fullBlkHamt.ForEach(fbc, func(key string) error {
		chainMessages, err := DecodeChainMessagesHAMT(ctx, store, fbc.BlsMessages)
		if err != nil {
			return err
		}
		signedChainMessages, err := DecodeSignedChainMessagesHAMT(ctx, store, fbc.SecpMessages)
		if err != nil {
			return err
		}
		bh := new(types.BlockHeader)
		*bh = *fbc.BlockHeader
		// TODO assert key == bh.Cid
		out[bh.Cid()] = &processor.FullBlock{
			Block:        bh,
			SecpMessages: signedChainMessages,
			BlsMessages:  chainMessages,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type ChainMessageIPLDContainer struct {
	Message       *types.Message                 `cborgen:"message"`
	Receipt       *processor.ChainMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                        `cborgen:"vm_messages"`
}

func MakeChainMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*processor.ChainMessage) (cid.Cid, error) {
	messageHamt, err := adt.MakeEmptyMap(store, 5)
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

func DecodeChainMessagesHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) ([]*processor.ChainMessage, error) {
	messagesHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}

	var out []*processor.ChainMessage
	mc := new(ChainMessageIPLDContainer)
	if err := messagesHamt.ForEach(mc, func(key string) error {
		msgs := *mc.Message
		rcpts := *mc.Receipt
		vmMessages, err := processor.VmMessageListFromAdtArray(store, mc.VmMessagesAmt, 5)
		if err != nil {
			return err
		}
		out = append(out, &processor.ChainMessage{
			Message:    &msgs,
			Receipt:    &rcpts,
			VmMessages: vmMessages,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type SignedChainMessageIPLDContainer struct {
	Message       *types.SignedMessage           `cborgen:"message"`
	Receipt       *processor.ChainMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                        `cborgen:"vm_messages"`
}

func MakeSignedChainMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*processor.SignedChainMessage) (cid.Cid, error) {
	messageHamt, err := adt.MakeEmptyMap(store, 5)
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

func DecodeSignedChainMessagesHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) ([]*processor.SignedChainMessage, error) {
	messagesHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}

	var out []*processor.SignedChainMessage
	mc := new(SignedChainMessageIPLDContainer)
	if err := messagesHamt.ForEach(mc, func(key string) error {
		msg := *mc.Message
		recet := *mc.Receipt
		vmMessages, err := processor.VmMessageListFromAdtArray(store, mc.VmMessagesAmt, 5)
		if err != nil {
			return err
		}
		out = append(out, &processor.SignedChainMessage{
			Message:    &msg,
			Receipt:    &recet,
			VmMessages: vmMessages,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type ImplicitMessageIPLDContainer struct {
	Message       *types.Message                    `cborgen:"message"`
	Receipt       *processor.ImplicitMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                           `cborgen:"vm_messages"`
}

// MakeImplicitMessagesHAMT returns the root of a hamt node containing the set of implicit messages
func MakeImplicitMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*processor.ImplicitMessage) (cid.Cid, error) {
	messageHamt, err := adt.MakeEmptyMap(store, 5)
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

func DecodeImplicitMessagesHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) ([]*processor.ImplicitMessage, error) {
	messagesHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}

	var out []*processor.ImplicitMessage
	msg := new(ImplicitMessageIPLDContainer)
	if err := messagesHamt.ForEach(msg, func(key string) error {
		m := *msg.Message
		rect := *msg.Receipt
		vmMessages, err := processor.VmMessageListFromAdtArray(store, msg.VmMessagesAmt, 5)
		if err != nil {
			return err
		}
		out = append(out, &processor.ImplicitMessage{
			Message:    &m,
			Receipt:    &rect,
			VmMessages: vmMessages,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

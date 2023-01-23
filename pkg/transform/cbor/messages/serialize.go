package messages

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	adtstore "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/chain"
)

type FullBlockIPLDContainer struct {
	BlockHeader  *types.BlockHeader `cborgen:"block_header"`
	SecpMessages cid.Cid            `cborgen:"secp_messages"`
	BlsMessages  cid.Cid            `cborgen:"bls_messages"`
}

func MakeFullBlockHAMT(ctx context.Context, store adtstore.Store, fullBlks map[cid.Cid]*chain.FullBlock) (cid.Cid, error) {
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

func DecodeFullBlockHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) (map[cid.Cid]*chain.FullBlock, error) {
	fullBlkHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}
	out := make(map[cid.Cid]*chain.FullBlock)
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
		out[bh.Cid()] = &chain.FullBlock{
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
	Message       *types.Message             `cborgen:"message"`
	Receipt       *chain.ChainMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                    `cborgen:"vm_messages"`
}

func MakeChainMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*chain.ChainMessage) (cid.Cid, error) {
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

func DecodeChainMessagesHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) ([]*chain.ChainMessage, error) {
	messagesHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}

	var out []*chain.ChainMessage
	mc := new(ChainMessageIPLDContainer)
	if err := messagesHamt.ForEach(mc, func(key string) error {
		var msg types.Message
		var rec chain.ChainMessageReceipt

		msg = *mc.Message
		if mc.Receipt != nil {
			rec = *mc.Receipt
		}
		vmMessages, err := chain.VmMessageListFromAdtArray(store, mc.VmMessagesAmt, 5)
		if err != nil {
			return err
		}
		out = append(out, &chain.ChainMessage{
			Message:    &msg,
			Receipt:    &rec,
			VmMessages: vmMessages,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type SignedChainMessageIPLDContainer struct {
	Message       *types.SignedMessage       `cborgen:"message"`
	Receipt       *chain.ChainMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                    `cborgen:"vm_messages"`
}

func MakeSignedChainMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*chain.SignedChainMessage) (cid.Cid, error) {
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

func DecodeSignedChainMessagesHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) ([]*chain.SignedChainMessage, error) {
	messagesHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}

	var out []*chain.SignedChainMessage
	mc := new(SignedChainMessageIPLDContainer)
	if err := messagesHamt.ForEach(mc, func(key string) error {
		var msg types.SignedMessage
		var rec chain.ChainMessageReceipt

		msg = *mc.Message
		if mc.Receipt != nil {
			rec = *mc.Receipt
		}
		vmMessages, err := chain.VmMessageListFromAdtArray(store, mc.VmMessagesAmt, 5)
		if err != nil {
			return err
		}
		out = append(out, &chain.SignedChainMessage{
			Message:    &msg,
			Receipt:    &rec,
			VmMessages: vmMessages,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type ImplicitMessageIPLDContainer struct {
	Message       *types.Message                `cborgen:"message"`
	Receipt       *chain.ImplicitMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                       `cborgen:"vm_messages"`
}

// MakeImplicitMessagesHAMT returns the root of a hamt node containing the set of implicit messages
func MakeImplicitMessagesHAMT(ctx context.Context, store adtstore.Store, messages []*chain.ImplicitMessage) (cid.Cid, error) {
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

func DecodeImplicitMessagesHAMT(ctx context.Context, store adtstore.Store, root cid.Cid) ([]*chain.ImplicitMessage, error) {
	messagesHamt, err := adt.AsMap(store, root, 5)
	if err != nil {
		return nil, err
	}

	var out []*chain.ImplicitMessage
	msg := new(ImplicitMessageIPLDContainer)
	if err := messagesHamt.ForEach(msg, func(key string) error {
		m := *msg.Message
		rect := *msg.Receipt
		vmMessages, err := chain.VmMessageListFromAdtArray(store, msg.VmMessagesAmt, 5)
		if err != nil {
			return err
		}
		out = append(out, &chain.ImplicitMessage{
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

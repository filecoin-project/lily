package messages

import (
	"context"
	"io"

	"github.com/filecoin-project/go-state-types/abi"
	adtstore "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/processor"
)

type MessageIPLDContainer struct {
	CurrentTipSet    *types.TipSet   `cborgen:"current"`
	ExecutedTipSet   *types.TipSet   `cborgen:"executed"`
	BaseFee          abi.TokenAmount `cborgen:"base_fee"`
	FullBlocks       cid.Cid         `cborgen:"full_blocks"`       // HAMT[blkCid]FullBlockIPLDContainer
	ImplicitMessages cid.Cid         `cborgen:"implicit_messages"` // HAMT[implicitCID]ImplicitMessageIPLDContainer
}

type FullBlockIPLDContainer struct {
	BlockHeader  *types.BlockHeader `cborgen:"block_header"`
	SecpMessages cid.Cid            `cborgen:"secp_messages"`
	BlsMessages  cid.Cid            `cborgen:"bls_messages"`
}

func (f *FullBlockIPLDContainer) MarshalCBOR(w io.Writer) error {
	//TODO implement me
	panic("implement me")
}

func ProcessMessages(ctx context.Context, store adtstore.Store, changes *processor.MessageStateChanges) (*MessageIPLDContainer, error) {
	fullBlkHamt, err := adt2.MakeEmptyMap(store, 5)
	if err != nil {
		return nil, err
	}
	for blkCid, fb := range changes.FullBlocks {
		blsMsgHamt, err := ProcessChainMessages(ctx, store, fb.BlsMessages)
		if err != nil {
			return nil, err
		}

		secpMsgHamt, err := ProcessSignedChainMessages(ctx, store, fb.SecpMessages)
		if err != nil {
			return nil, err
		}

		if err := fullBlkHamt.Put(abi.CidKey(blkCid), &FullBlockIPLDContainer{
			BlockHeader:  fb.Block,
			SecpMessages: secpMsgHamt,
			BlsMessages:  blsMsgHamt,
		}); err != nil {
			return nil, err
		}
	}

	implicitMsgHamt, err := ProcessImplicitMessages(ctx, store, changes.ImplicitMessages)
	if err != nil {
		return nil, err
	}

	fullBlkRoot, err := fullBlkHamt.Root()
	if err != nil {
		return nil, err
	}

	return &MessageIPLDContainer{
		CurrentTipSet:    changes.Current,
		ExecutedTipSet:   changes.Executed,
		BaseFee:          changes.BaseFee,
		FullBlocks:       fullBlkRoot,
		ImplicitMessages: implicitMsgHamt,
	}, nil
}

type ChainMessageIPLDContainer struct {
	Message       *types.Message                 `cborgen:"message"`
	Receipt       *processor.ChainMessageReceipt `cborgen:"receipt"`
	VmMessagesAmt cid.Cid                        `cborgen:"vm_messages"`
}

func (c *ChainMessageIPLDContainer) MarshalCBOR(w io.Writer) error {
	//TODO implement me
	panic("implement me")
}

func ProcessChainMessages(ctx context.Context, store adtstore.Store, messages []*processor.ChainMessage) (cid.Cid, error) {
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

func (s *SignedChainMessageIPLDContainer) MarshalCBOR(w io.Writer) error {
	//TODO implement me
	panic("implement me")
}

func ProcessSignedChainMessages(ctx context.Context, store adtstore.Store, messages []*processor.SignedChainMessage) (cid.Cid, error) {
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

func (i *ImplicitMessageIPLDContainer) MarshalCBOR(w io.Writer) error {
	//TODO implement me
	panic("implement me")
}

// ProcessImplicitMessages returns the root of a hamt node containing the set of implicit messages
func ProcessImplicitMessages(ctx context.Context, store adtstore.Store, messages []*processor.ImplicitMessage) (cid.Cid, error) {
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

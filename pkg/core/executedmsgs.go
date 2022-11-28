package core

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/tasks"
)

// ExecutedMessage are message that were executed while applying a block.
type ExecutedMessage struct {
	Meta *MessageMeta

	// Message
	Message *types.Message

	// Receipt
	ReceiptIdx int64
	Receipt    *types.MessageReceipt

	// GasOutputs
	GasOutputs *GasOutputs
}

type MessageMeta struct {
	MessageCID cid.Cid
	Signature  *crypto.Signature
	SizeBytes  int64
}

type GasOutputs struct {
	BaseFeeBurn        abi.TokenAmount
	OverEstimationBurn abi.TokenAmount
	MinerPenalty       abi.TokenAmount
	MinerTip           abi.TokenAmount
	Refund             abi.TokenAmount
	GasRefund          int64
	GasBurned          int64
}

func ExtractExecutedMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]*ExecutedMessage, error) {
	blkMsgRec, err := api.TipSetMessageReceipts(ctx, current, executed)
	if err != nil {
		return nil, fmt.Errorf("getting messages and receipts: %w", err)
	}

	burnFn, err := api.ShouldBurnFn(ctx, executed)
	if err != nil {
		return nil, fmt.Errorf("getting should burn function: %w", err)
	}

	var out = make([]*ExecutedMessage, 0, len(blkMsgRec))
	for _, msgrec := range blkMsgRec {
		itr, err := msgrec.Iterator()
		if err != nil {
			return nil, err
		}

		for itr.HasNext() {
			msg, recIdx, rec := itr.Next()

			gasOutputs, err := datasource.ComputeGasOutputs(ctx, msgrec.Block, msg.VMMessage(), rec, burnFn)
			if err != nil {
				return nil, fmt.Errorf("failed to compute gas outputs: %w", err)
			}

			var sig *crypto.Signature
			sm, ok := msg.(*types.SignedMessage)
			if ok {
				sig = &sm.Signature
			}
			out = append(out, &ExecutedMessage{
				Meta: &MessageMeta{
					MessageCID: msg.Cid(),
					Signature:  sig,
					SizeBytes:  int64(msg.ChainLength()),
				},
				Message: msg.VMMessage(),

				ReceiptIdx: int64(recIdx),
				Receipt:    rec,

				GasOutputs: &GasOutputs{
					BaseFeeBurn:        gasOutputs.BaseFeeBurn,
					OverEstimationBurn: gasOutputs.OverEstimationBurn,
					MinerPenalty:       gasOutputs.MinerPenalty,
					MinerTip:           gasOutputs.MinerTip,
					Refund:             gasOutputs.Refund,
					GasRefund:          gasOutputs.GasRefund,
					GasBurned:          gasOutputs.GasBurned,
				},
			})
		}
	}
	return out, nil
}

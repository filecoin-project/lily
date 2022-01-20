package messages

import (
	"bytes"
	"context"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/model"
	derivedmodel "github.com/filecoin-project/lily/model/derived"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lily/tasks/messages/fcjson"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"golang.org/x/xerrors"
	"math"
	"math/big"
)

func init() {
	model.RegisterTipSetModelExtractor(&messagemodel.Message{}, MessageExtractor{})
	model.RegisterTipSetModelExtractor(&messagemodel.Receipt{}, ReceiptExtractor{})
	model.RegisterTipSetModelExtractor(&messagemodel.ParsedMessage{}, ParsedMessageExtractor{})
	model.RegisterTipSetModelExtractor(&messagemodel.MessageGasEconomy{}, MessageGasEconomicsExtractor{})
	model.RegisterTipSetModelExtractor(&derivedmodel.GasOutputs{}, GasOutputExtractor{})
	model.RegisterTipSetModelExtractor(&messagemodel.BlockMessage{}, BlockMessageExtractor{})
}

var _ model.TipSetStateExtractor = (*MessageExtractor)(nil)

type MessageExtractor struct{}

func (MessageExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	res, err := process(ctx, current, previous, api)
	return res.Messages, err
}

func (MessageExtractor) Name() string {
	return "messages"
}

var _ model.TipSetStateExtractor = (*ReceiptExtractor)(nil)

type ReceiptExtractor struct{}

func (ReceiptExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	res, err := process(ctx, current, previous, api)
	return res.Receipts, err
}

func (ReceiptExtractor) Name() string {
	return "receipts"
}

var _ model.TipSetStateExtractor = (*ParsedMessageExtractor)(nil)

type ParsedMessageExtractor struct{}

func (ParsedMessageExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	res, err := process(ctx, current, previous, api)
	return res.ParsedMsgs, err
}

func (ParsedMessageExtractor) Name() string {
	return "parsed_messages"
}

var _ model.TipSetStateExtractor = (*GasOutputExtractor)(nil)

type GasOutputExtractor struct{}

func (GasOutputExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	res, err := process(ctx, current, previous, api)
	return res.GasOutputs, err
}

func (GasOutputExtractor) Name() string {
	return "derived_gas_outputs"
}

var _ model.TipSetStateExtractor = (*MessageGasEconomicsExtractor)(nil)

type MessageGasEconomicsExtractor struct{}

func (MessageGasEconomicsExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	res, err := process(ctx, current, previous, api)
	return res.MsgGasEcon, err
}

func (MessageGasEconomicsExtractor) Name() string {
	return "messages_gas_economy"
}

var _ model.TipSetStateExtractor = (*BlockMessageExtractor)(nil)

type BlockMessageExtractor struct{}

func (BlockMessageExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	res, err := process(ctx, current, previous, api)
	return res.BlockMsgs, err
}

func (BlockMessageExtractor) Name() string {
	return "block_messages"
}

type processResult struct {
	Messages   messagemodel.Messages
	BlockMsgs  messagemodel.BlockMessages
	Receipts   messagemodel.Receipts
	ParsedMsgs messagemodel.ParsedMessages
	GasOutputs derivedmodel.GasOutputsList
	MsgGasEcon *messagemodel.MessageGasEconomy
}

func process(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (*processResult, error) {
	tsMsgs, err := api.GetExecutedAndBlockMessagesForTipset(ctx, current, previous)
	if err != nil {
		return nil, err
	}
	emsgs := tsMsgs.Executed
	blkMsgs := tsMsgs.Block
	var (
		messageResults       = make(messagemodel.Messages, 0, len(emsgs))
		receiptResults       = make(messagemodel.Receipts, 0, len(emsgs))
		parsedMessageResults = make(messagemodel.ParsedMessages, 0, len(emsgs))
		gasOutputsResults    = make(derivedmodel.GasOutputsList, 0, len(emsgs))
		errorsDetected       = make([]*MessageError, 0, len(emsgs))
	)

	var (
		exeMsgSeen        = make(map[cid.Cid]bool, len(emsgs))
		blkMsgSeen        = make(map[cid.Cid]bool)
		totalGasLimit     int64
		totalUniqGasLimit int64
	)

	// Record which blocks had which messages, regardless of duplicates
	blockMessageResults := messagemodel.BlockMessages{}
	for _, bm := range blkMsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		blk := bm.Block
		for _, msg := range bm.SecpMessages {
			blockMessageResults = append(blockMessageResults, &messagemodel.BlockMessage{
				Height:  int64(bm.Block.Height),
				Block:   blk.Cid().String(),
				Message: msg.Cid().String(),
			})

			if blkMsgSeen[msg.Cid()] {
				continue
			}
			blkMsgSeen[msg.Cid()] = true

			var msgSize int
			if b, err := msg.Message.Serialize(); err == nil {
				msgSize = len(b)
			} else {
				errorsDetected = append(errorsDetected, &MessageError{
					Cid:   msg.Cid(),
					Error: xerrors.Errorf("failed to serialize message: %w", err).Error(),
				})
			}

			// record all unique Secp messages
			msg := &messagemodel.Message{
				Height:     int64(bm.Block.Height),
				Cid:        msg.Cid().String(),
				From:       msg.Message.From.String(),
				To:         msg.Message.To.String(),
				Value:      msg.Message.Value.String(),
				GasFeeCap:  msg.Message.GasFeeCap.String(),
				GasPremium: msg.Message.GasPremium.String(),
				GasLimit:   msg.Message.GasLimit,
				SizeBytes:  msgSize,
				Nonce:      msg.Message.Nonce,
				Method:     uint64(msg.Message.Method),
			}
			messageResults = append(messageResults, msg)

		}
		for _, msg := range bm.BlsMessages {
			blockMessageResults = append(blockMessageResults, &messagemodel.BlockMessage{
				Height:  int64(bm.Block.Height),
				Block:   blk.Cid().String(),
				Message: msg.Cid().String(),
			})

			if blkMsgSeen[msg.Cid()] {
				continue
			}
			blkMsgSeen[msg.Cid()] = true

			var msgSize int
			if b, err := msg.Serialize(); err == nil {
				msgSize = len(b)
			} else {
				errorsDetected = append(errorsDetected, &MessageError{
					Cid:   msg.Cid(),
					Error: xerrors.Errorf("failed to serialize message: %w", err).Error(),
				})
			}

			// record all unique bls messages
			msg := &messagemodel.Message{
				Height:     int64(bm.Block.Height),
				Cid:        msg.Cid().String(),
				From:       msg.From.String(),
				To:         msg.To.String(),
				Value:      msg.Value.String(),
				GasFeeCap:  msg.GasFeeCap.String(),
				GasPremium: msg.GasPremium.String(),
				GasLimit:   msg.GasLimit,
				SizeBytes:  msgSize,
				Nonce:      msg.Nonce,
				Method:     uint64(msg.Method),
			}
			messageResults = append(messageResults, msg)
		}
	}

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		// calculate total gas limit of executed messages regardless of duplicates.
		for range m.Blocks {
			totalGasLimit += m.Message.GasLimit
		}

		if exeMsgSeen[m.Cid] {
			continue
		}
		exeMsgSeen[m.Cid] = true
		totalUniqGasLimit += m.Message.GasLimit

		var msgSize int
		if b, err := m.Message.Serialize(); err == nil {
			msgSize = len(b)
		} else {
			errorsDetected = append(errorsDetected, &MessageError{
				Cid:   m.Cid,
				Error: xerrors.Errorf("failed to serialize message: %w", err).Error(),
			})
		}

		rcpt := &messagemodel.Receipt{
			Height:    int64(current.Height()),
			Message:   m.Cid.String(),
			StateRoot: current.ParentState().String(),
			Idx:       int(m.Index),
			ExitCode:  int64(m.Receipt.ExitCode),
			GasUsed:   m.Receipt.GasUsed,
		}
		receiptResults = append(receiptResults, rcpt)

		actorName := builtin.ActorNameByCode(m.ToActorCode)
		gasOutput := &derivedmodel.GasOutputs{
			Height:             int64(m.Height),
			Cid:                m.Cid.String(),
			From:               m.Message.From.String(),
			To:                 m.Message.To.String(),
			Value:              m.Message.Value.String(),
			GasFeeCap:          m.Message.GasFeeCap.String(),
			GasPremium:         m.Message.GasPremium.String(),
			GasLimit:           m.Message.GasLimit,
			Nonce:              m.Message.Nonce,
			Method:             uint64(m.Message.Method),
			StateRoot:          m.BlockHeader.ParentStateRoot.String(),
			ExitCode:           rcpt.ExitCode,
			GasUsed:            rcpt.GasUsed,
			ParentBaseFee:      m.BlockHeader.ParentBaseFee.String(),
			SizeBytes:          msgSize,
			BaseFeeBurn:        m.GasOutputs.BaseFeeBurn.String(),
			OverEstimationBurn: m.GasOutputs.OverEstimationBurn.String(),
			MinerPenalty:       m.GasOutputs.MinerPenalty.String(),
			MinerTip:           m.GasOutputs.MinerTip.String(),
			Refund:             m.GasOutputs.Refund.String(),
			GasRefund:          m.GasOutputs.GasRefund,
			GasBurned:          m.GasOutputs.GasBurned,
			ActorName:          actorName,
			ActorFamily:        builtin.ActorFamily(actorName),
		}
		gasOutputsResults = append(gasOutputsResults, gasOutput)

		if m.ToActorCode.Defined() {
			method, params, err := parseMessageParams(m.Message, m.ToActorCode)
			if err == nil {
				pm := &messagemodel.ParsedMessage{
					Height: int64(m.Height),
					Cid:    m.Cid.String(),
					From:   m.Message.From.String(),
					To:     m.Message.To.String(),
					Value:  m.Message.Value.String(),
					Method: method,
					Params: params,
				}
				parsedMessageResults = append(parsedMessageResults, pm)
			} else {
				if rcpt.ExitCode == int64(exitcode.ErrSerialization) || rcpt.ExitCode == int64(exitcode.ErrIllegalArgument) || rcpt.ExitCode == int64(exitcode.SysErrInvalidMethod) {
					// ignore the parse error since the params are probably malformed, as reported by the vm
				} else {
					errorsDetected = append(errorsDetected, &MessageError{
						Cid:   m.Cid,
						Error: xerrors.Errorf("failed to parse message params: %w", err).Error(),
					})
				}
			}
		} else {
			// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
			// message fails then Lotus will not allow the actor to be created and we are left with an address of an
			// unknown type.
			// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
			// indicates that Lily actor type detection is out of date.
			if rcpt.ExitCode == 0 {
				errorsDetected = append(errorsDetected, &MessageError{
					Cid:   m.Cid,
					Error: xerrors.Errorf("failed to parse message params: missing to actor code").Error(),
				})
			}
		}
	}

	newBaseFee := store.ComputeNextBaseFee(previous.Blocks()[0].ParentBaseFee, totalUniqGasLimit, len(previous.Blocks()), previous.Height())
	baseFeeRat := new(big.Rat).SetFrac(newBaseFee.Int, new(big.Int).SetUint64(build.FilecoinPrecision))
	baseFee, _ := baseFeeRat.Float64()

	baseFeeChange := new(big.Rat).SetFrac(newBaseFee.Int, previous.Blocks()[0].ParentBaseFee.Int)
	baseFeeChangeF, _ := baseFeeChange.Float64()

	messageGasEconomyResult := &messagemodel.MessageGasEconomy{
		Height:              int64(previous.Height()),
		StateRoot:           previous.ParentState().String(),
		GasLimitTotal:       totalGasLimit,
		GasLimitUniqueTotal: totalUniqGasLimit,
		BaseFee:             baseFee,
		BaseFeeChangeLog:    math.Log(baseFeeChangeF) / math.Log(1.125),
		GasFillRatio:        float64(totalGasLimit) / float64(len(previous.Blocks())*build.BlockGasTarget),
		GasCapacityRatio:    float64(totalUniqGasLimit) / float64(len(previous.Blocks())*build.BlockGasTarget),
		GasWasteRatio:       float64(totalGasLimit-totalUniqGasLimit) / float64(len(previous.Blocks())*build.BlockGasTarget),
	}

	return &processResult{
		Messages:   messageResults,
		BlockMsgs:  blockMessageResults,
		Receipts:   receiptResults,
		ParsedMsgs: parsedMessageResults,
		GasOutputs: gasOutputsResults,
		MsgGasEcon: messageGasEconomyResult,
	}, nil

}

func parseMessageParams(m *types.Message, destCode cid.Cid) (string, string, error) {
	// Method is optional, zero means a plain value transfer
	if m.Method == 0 {
		return "Send", "", nil
	}

	if !destCode.Defined() {
		return "Unknown", "", xerrors.Errorf("missing actor code")
	}

	var params ipld.Node
	var method string
	var err error

	params, method, err = ParseParams(m.Params, int64(m.Method), destCode)
	if method == "Unknown" {
		return "", "", xerrors.Errorf("unknown method for actor type %s: %d", destCode.String(), int64(m.Method))
	}
	if err != nil {
		log.Warnf("failed to parse parameters of message %s: %v", m.Cid, err)
		// this can occur when the message is not valid cbor
		return method, "", err
	}
	if params == nil {
		return method, "", nil
	}

	buf := bytes.NewBuffer(nil)
	if err := fcjson.Encoder(params, buf); err != nil {
		return "", "", xerrors.Errorf("json encode: %w", err)
	}

	encoded := string(bytes.ReplaceAll(bytes.ToValidUTF8(buf.Bytes(), []byte{}), []byte{0x00}, []byte{}))

	return method, encoded, nil

}

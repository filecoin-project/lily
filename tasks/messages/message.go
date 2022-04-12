package messages

import (
	"context"
	"math"
	"math/big"

	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/model"
	derivedmodel "github.com/filecoin-project/lily/model/derived"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

// Note that pts is the parent tipset containing the messages, ts is the following tipset containing the receipts
func (p *Task) ProcessMessages(ctx context.Context, ts *types.TipSet, pts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessMessages")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.String()), attribute.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(pts.Height()),
		StateRoot: pts.ParentState().String(),
	}

	tsMsgs, err := p.node.ExecutedAndBlockMessages(ctx, ts, pts)
	if err != nil {
		report.ErrorsDetected = xerrors.Errorf("getting executed and block messages: %w", err)
		return nil, report, nil
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
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
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
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
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
			Height:    int64(ts.Height()),
			Message:   m.Cid.String(),
			StateRoot: ts.ParentState().String(),
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
			method, params, err := p.parseMessageParams(m.Message, m.ToActorCode)
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

	newBaseFee := store.ComputeNextBaseFee(pts.Blocks()[0].ParentBaseFee, totalUniqGasLimit, len(pts.Blocks()), pts.Height())
	baseFeeRat := new(big.Rat).SetFrac(newBaseFee.Int, new(big.Int).SetUint64(build.FilecoinPrecision))
	baseFee, _ := baseFeeRat.Float64()

	baseFeeChange := new(big.Rat).SetFrac(newBaseFee.Int, pts.Blocks()[0].ParentBaseFee.Int)
	baseFeeChangeF, _ := baseFeeChange.Float64()

	messageGasEconomyResult := &messagemodel.MessageGasEconomy{
		Height:              int64(pts.Height()),
		StateRoot:           pts.ParentState().String(),
		GasLimitTotal:       totalGasLimit,
		GasLimitUniqueTotal: totalUniqGasLimit,
		BaseFee:             baseFee,
		BaseFeeChangeLog:    math.Log(baseFeeChangeF) / math.Log(1.125),
		GasFillRatio:        float64(totalGasLimit) / float64(len(pts.Blocks())*build.BlockGasTarget),
		GasCapacityRatio:    float64(totalUniqGasLimit) / float64(len(pts.Blocks())*build.BlockGasTarget),
		GasWasteRatio:       float64(totalGasLimit-totalUniqGasLimit) / float64(len(pts.Blocks())*build.BlockGasTarget),
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		messageResults,
		receiptResults,
		blockMessageResults,
		parsedMessageResults,
		gasOutputsResults,
		messageGasEconomyResult,
	}, report, nil
}

func (p *Task) parseMessageParams(m *types.Message, destCode cid.Cid) (string, string, error) {
	return util.MethodAndParamsForMessage(m, destCode)
}

type MessageError struct {
	Cid   cid.Cid
	Error string
}

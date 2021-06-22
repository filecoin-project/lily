package messages

import (
	"bytes"
	"context"
	"math"
	"math/big"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	derivedmodel "github.com/filecoin-project/sentinel-visor/model/derived"
	messagemodel "github.com/filecoin-project/sentinel-visor/model/messages"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/tasks/messages/fcjson"
)

var log = logging.Logger("visor/task/messages")

type Task struct {
}

func NewTask() *Task {
	return &Task{}
}

func (p *Task) ProcessMessages(ctx context.Context, ts *types.TipSet, pts *types.TipSet, emsgs []*lens.ExecutedMessage, blkMsgs []*lens.BlockMessages) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := global.Tracer("").Start(ctx, "ProcessMessages")
	if span.IsRecording() {
		span.SetAttributes(label.String("tipset", ts.String()), label.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(pts.Height()),
		StateRoot: pts.ParentState().String(),
	}

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
				Height:  int64(ts.Height()),
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
				Height:     int64(ts.Height()),
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
				Height:  int64(ts.Height()),
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
				Height:     int64(ts.Height()),
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
			Height:    int64(ts.Height()), // this is the child height
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
			errorsDetected = append(errorsDetected, &MessageError{
				Cid:   m.Cid,
				Error: xerrors.Errorf("failed to parse message params: %w", err).Error(),
			})
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
	// Method is optional, zero means a plain value transfer
	if m.Method == 0 {
		return "Send", "", nil
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

func (p *Task) Close() error {
	return nil
}

type MessageError struct {
	Cid   cid.Cid
	Error string
}

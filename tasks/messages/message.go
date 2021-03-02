package messages

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/statediff"
	"github.com/filecoin-project/statediff/codec/fcjson"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	derivedmodel "github.com/filecoin-project/sentinel-visor/model/derived"
	messagemodel "github.com/filecoin-project/sentinel-visor/model/messages"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
)

var log = logging.Logger("messages")

type Task struct {
	node   lens.API
	opener lens.APIOpener
	closer lens.APICloser
}

func NewTask(opener lens.APIOpener) *Task {
	return &Task{
		opener: opener,
	}
}

func (p *Task) ProcessMessages(ctx context.Context, ts *types.TipSet, pts *types.TipSet, emsgs []*lens.ExecutedMessage) (model.Persistable, *visormodel.ProcessingReport, error) {
	if p.node == nil {
		node, closer, err := p.opener.Open(ctx)
		if err != nil {
			return nil, nil, xerrors.Errorf("unable to open lens: %w", err)
		}
		p.node = node
		p.closer = closer
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(pts.Height()),
		StateRoot: pts.ParentState().String(),
	}

	var (
		messageResults       = make(messagemodel.Messages, 0, len(emsgs))
		receiptResults       = make(messagemodel.Receipts, 0, len(emsgs))
		blockMessageResults  = make(messagemodel.BlockMessages, 0, len(emsgs))
		parsedMessageResults = make(messagemodel.ParsedMessages, 0, len(emsgs))
		gasOutputsResults    = make(derivedmodel.GasOutputsList, 0, len(emsgs))
		errorsDetected       = make([]*MessageError, 0, len(emsgs))
	)

	var (
		seen = make(map[cid.Cid]bool, len(emsgs))

		totalGasLimit     int64
		totalUniqGasLimit int64
	)

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		// Record which blocks had which messages, regardless of duplicates
		for _, blockCid := range m.Blocks {
			blockMessageResults = append(blockMessageResults, &messagemodel.BlockMessage{
				Height:  int64(m.Height),
				Block:   blockCid.String(),
				Message: m.Cid.String(),
			})
			totalGasLimit += m.Message.GasLimit
		}

		if seen[m.Cid] {
			continue
		}
		seen[m.Cid] = true
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

		// record all unique messages
		msg := &messagemodel.Message{
			Height:     int64(m.Height),
			Cid:        m.Cid.String(),
			From:       m.Message.From.String(),
			To:         m.Message.To.String(),
			Value:      m.Message.Value.String(),
			GasFeeCap:  m.Message.GasFeeCap.String(),
			GasPremium: m.Message.GasPremium.String(),
			GasLimit:   m.Message.GasLimit,
			SizeBytes:  msgSize,
			Nonce:      m.Message.Nonce,
			Method:     uint64(m.Message.Method),
		}
		messageResults = append(messageResults, msg)

		rcpt := &messagemodel.Receipt{
			Height:    int64(ts.Height()), // this is the child height
			Message:   msg.Cid,
			StateRoot: ts.ParentState().String(),
			Idx:       int(m.Index),
			ExitCode:  int64(m.Receipt.ExitCode),
			GasUsed:   m.Receipt.GasUsed,
		}
		receiptResults = append(receiptResults, rcpt)

		gasOutput := &derivedmodel.GasOutputs{
			Height:             msg.Height,
			Cid:                msg.Cid,
			From:               msg.From,
			To:                 msg.To,
			Value:              msg.Value,
			GasFeeCap:          msg.GasFeeCap,
			GasPremium:         msg.GasPremium,
			GasLimit:           msg.GasLimit,
			Nonce:              msg.Nonce,
			Method:             msg.Method,
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
			ActorName:          actorstate.ActorNameByCode(m.ToActorCode),
		}
		gasOutputsResults = append(gasOutputsResults, gasOutput)

		method, params, err := p.parseMessageParams(m.Message, m.ToActorCode)
		if err == nil {
			pm := &messagemodel.ParsedMessage{
				Height: msg.Height,
				Cid:    msg.Cid,
				From:   msg.From,
				To:     msg.To,
				Value:  msg.Value,
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
	actor, ok := statediff.LotusActorCodes[destCode.String()]
	if !ok {
		actor = statediff.LotusTypeUnknown
	}
	var params ipld.Node
	var method string
	var err error

	// TODO: the following closure is in place to handle the potential for panic
	// in ipld-prime. Can be removed once fixed upstream.
	// tracking issue: https://github.com/ipld/go-ipld-prime/issues/97
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = xerrors.Errorf("recovered panic: %v", r)
			}
		}()
		params, method, err = statediff.ParseParams(m.Params, int64(m.Method), actor)
	}()
	if err != nil && actor != statediff.LotusTypeUnknown {
		// fall back to generic cbor->json conversion.
		actor = statediff.LotusTypeUnknown
		params, method, err = statediff.ParseParams(m.Params, int64(m.Method), actor)
	}
	if method == "Unknown" {
		method = fmt.Sprintf("%s.%d", actor, m.Method)
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
	if p.closer != nil {
		p.closer()
		p.closer = nil
	}
	p.node = nil
	return nil
}

type MessageError struct {
	Cid   cid.Cid
	Error string
}

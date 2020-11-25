package message

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime"
	"github.com/raulk/clock"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	messagemodel "github.com/filecoin-project/sentinel-visor/model/messages"
	"github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/wait"
	"github.com/filecoin-project/statediff"
	"github.com/filecoin-project/statediff/codec/fcjson"
)

const (
	idleSleepInterval = 60 * time.Second       // time to wait if the processor runs out of blocks to process
	batchInterval     = 100 * time.Millisecond // time to wait between batches
)

var (
	accountActorCodeID string
	log                = logging.Logger("message")
)

func init() {
	for code, actor := range statediff.LotusActorCodes {
		if actor == statediff.AccountActorState {
			accountActorCodeID = code
			break
		}
	}
}

func NewMessageProcessor(d *storage.Database, opener lens.APIOpener, leaseLength time.Duration, batchSize int, parseMessages bool, minHeight, maxHeight int64) *MessageProcessor {
	return &MessageProcessor{
		opener:        opener,
		storage:       d,
		leaseLength:   leaseLength,
		batchSize:     batchSize,
		parseMessages: parseMessages,
		minHeight:     minHeight,
		maxHeight:     maxHeight,
		clock:         clock.New(),
	}
}

// MessageProcessor is a task that processes blocks to detect messages and persists
// their details to the database.
type MessageProcessor struct {
	opener        lens.APIOpener
	storage       *storage.Database
	leaseLength   time.Duration // length of time to lease work for
	batchSize     int           // number of tipsets to lease in a batch
	parseMessages bool          // if derived parsed messages should be calculated
	minHeight     int64         // limit processing to tipsets equal to or above this height
	maxHeight     int64         // limit processing to tipsets equal to or below this height
	clock         clock.Clock
}

// Run starts processing batches of tipsets and blocks until the context is done or
// an error occurs.
func (p *MessageProcessor) Run(ctx context.Context) error {
	node, closer, err := p.opener.Open(ctx)
	if err != nil {
		return xerrors.Errorf("open lens: %w", err)
	}
	defer closer()

	// TODO: restart delay when error returned

	// Loop until context is done or processing encounters a fatal error
	return wait.RepeatUntil(ctx, batchInterval, func(ctx context.Context) (bool, error) {
		return p.processBatch(ctx, node)
	})
}

func (p *MessageProcessor) processBatch(ctx context.Context, node lens.API) (bool, error) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "message"))
	ctx, span := global.Tracer("").Start(ctx, "MessageProcessor.processBatch")
	defer span.End()

	claimUntil := p.clock.Now().Add(p.leaseLength)

	// Lease some blocks to work on
	batch, err := p.storage.LeaseTipSetMessages(ctx, claimUntil, p.batchSize, p.minHeight, p.maxHeight)
	if err != nil {
		return false, xerrors.Errorf("lease tipset messages: %w", err)
	}

	// If we have no tipsets to work on then wait before trying again
	if len(batch) == 0 {
		sleepInterval := wait.Jitter(idleSleepInterval, 2)
		log.Debugf("no tipsets to process, waiting for %s", sleepInterval)
		time.Sleep(sleepInterval)
		return false, nil
	}

	log.Debugw("leased batch of tipsets", "count", len(batch))
	ctx, cancel := context.WithDeadline(ctx, claimUntil)
	defer cancel()

	for _, item := range batch {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return false, nil // Don't propagate cancelation error so we can resume processing cleanly
		default:
		}

		if err := p.processItem(ctx, node, item); err != nil {
			// Any errors are likely to be problems using the lens, mark this tipset as failed and exit this batch
			log.Errorw("failed to process tipset", "error", err.Error(), "height", item.Height)
			if err := p.storage.MarkTipSetMessagesComplete(ctx, item.TipSet, item.Height, p.clock.Now(), err.Error()); err != nil {
				log.Errorw("failed to mark tipset messages complete", "error", err.Error(), "height", item.Height)
			}

			return false, xerrors.Errorf("process item: %w", err)
		}

		if err := p.storage.MarkTipSetMessagesComplete(ctx, item.TipSet, item.Height, p.clock.Now(), ""); err != nil {
			log.Errorw("failed to mark tipset message complete", "error", err.Error(), "height", item.Height)
		}
	}

	return false, nil
}

func (p *MessageProcessor) processItem(ctx context.Context, node lens.API, item *visor.ProcessingTipSet) error {
	ctx, span := global.Tracer("").Start(ctx, "MessageProcessor.processItem")
	defer span.End()
	span.SetAttributes(label.Any("height", item.Height), label.Any("tipset", item.TipSet))

	stats.Record(ctx, metrics.TipsetHeight.M(item.Height))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	tsk, err := item.TipSetKey()
	if err != nil {
		return xerrors.Errorf("get tipsetkey: %w", err)
	}

	ts, err := node.ChainGetTipSet(ctx, tsk)
	if err != nil {
		return xerrors.Errorf("get tipset: %w", err)
	}

	if err := p.processTipSet(ctx, node, ts); err != nil {
		return xerrors.Errorf("process tipset: %w", err)
	}

	return nil
}

func (p *MessageProcessor) processTipSet(ctx context.Context, node lens.API, ts *types.TipSet) error {
	ctx, span := global.Tracer("").Start(ctx, "MessageProcessor.processTipSet")
	defer span.End()

	ll := log.With("height", int64(ts.Height()))

	blkMsgs, err := p.fetchMessages(ctx, node, ts)
	if err != nil {
		return xerrors.Errorf("fetch messages: %w", err)
	}

	rcts, err := p.fetchReceipts(ctx, node, ts)
	if err != nil {
		return xerrors.Errorf("fetch receipts: %w", err)
	}

	result, processingMsgs, err := p.extractMessageModels(ctx, node, ts, blkMsgs)
	if err != nil {
		return xerrors.Errorf("extract message models: %w", err)
	}
	result.Receipts = rcts

	ll.Debugw("persisting tipset", "messages", len(result.Messages), "block_messages", len(result.BlockMessages), "receipts", len(rcts))

	if err := p.storage.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := result.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		if err := processingMsgs.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return xerrors.Errorf("persist: %w", err)
	}

	return nil
}

func (p *MessageProcessor) fetchMessages(ctx context.Context, node lens.API, ts *types.TipSet) (map[cid.Cid]*api.BlockMessages, error) {
	out := make(map[cid.Cid]*api.BlockMessages)
	for _, blk := range ts.Cids() {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}
		blkMsgs, err := node.ChainGetBlockMessages(ctx, blk)
		if err != nil {
			return nil, xerrors.Errorf("get block messages: %w", err)
		}
		out[blk] = blkMsgs
	}
	return out, nil
}

func (p *MessageProcessor) extractMessageModels(ctx context.Context, node lens.API, ts *types.TipSet, blkMsgs map[cid.Cid]*api.BlockMessages) (*messagemodel.MessageTaskResult, visor.ProcessingMessageList, error) {
	result := &messagemodel.MessageTaskResult{
		Messages:          messagemodel.Messages{},
		BlockMessages:     messagemodel.BlockMessages{},
		ParsedMessages:    messagemodel.ParsedMessages{},
		MessageGasEconomy: nil,
	}

	pmsgModels := visor.ProcessingMessageList{}

	msgsSeen := map[cid.Cid]struct{}{}
	totalGasLimit := int64(0)
	totalUniqGasLimit := int64(0)

	for blk, msgs := range blkMsgs {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		// extract all messages, vmm will include duplicate messages.
		vmm := make([]*types.Message, 0, len(msgs.Cids))
		for _, m := range msgs.BlsMessages {
			vmm = append(vmm, m)
		}
		for _, m := range msgs.SecpkMessages {
			vmm = append(vmm, &m.Message)
		}

		for _, message := range vmm {
			// record which blocks had which messages
			result.BlockMessages = append(result.BlockMessages, &messagemodel.BlockMessage{
				Height:  int64(ts.Height()),
				Block:   blk.String(),
				Message: message.Cid().String(),
			})

			totalUniqGasLimit += message.GasLimit
			if _, seen := msgsSeen[message.Cid()]; seen {
				continue
			}
			totalGasLimit += message.GasLimit

			// record this message for processing by later stages
			pmsgModels = append(pmsgModels, visor.NewProcessingMessage(message, int64(ts.Height())))

			var msgSize int
			if b, err := message.Serialize(); err == nil {
				msgSize = len(b)
			} else {
				return nil, nil, xerrors.Errorf("serialize message: %w", err)
			}

			// record all unique messages
			msg := &messagemodel.Message{
				Height:     int64(ts.Height()),
				Cid:        message.Cid().String(),
				From:       message.From.String(),
				To:         message.To.String(),
				Value:      message.Value.String(),
				GasFeeCap:  message.GasFeeCap.String(),
				GasPremium: message.GasPremium.String(),
				GasLimit:   message.GasLimit,
				SizeBytes:  msgSize,
				Nonce:      message.Nonce,
				Method:     uint64(message.Method),
			}
			result.Messages = append(result.Messages, msg)

			msgsSeen[message.Cid()] = struct{}{}

			if p.parseMessages {
				dstAddr, err := address.NewFromString(msg.To)
				if err != nil {
					return nil, nil, xerrors.Errorf("parse to address failed for %s: %w", message.Cid().String(), err)
				}

				child, err := node.ChainGetTipSetByHeight(ctx, ts.Height()+1, types.NewTipSetKey())
				if err != nil {
					// If we aren't finalized, we fail for now, because a child tipset may occur
					if head, headErr := node.ChainHead(ctx); headErr == nil && head.Height()-ts.Height() < build.Finality {
						log.Warnf("Delaying derivation for message %s which is not yet finalized", message.Cid().String())
						return nil, nil, xerrors.Errorf("Failed to load child tipset: %w", err)
					}
					log.Infof("Skipping derivation of message parameters for message %s with no children blocks after derivation.", message.Cid().String())
					continue
				}
				if !cidsEqual(child.Parents().Cids(), ts.Cids()) {
					// if we aren't on the main chain, we don't have an easy way to get child blocks, so skip parsing these messages for now.
					log.Infof("Skipping derivation of message parameters for message %s not on canonical chain", message.Cid().String())
					continue
				}

				st, err := state.LoadStateTree(node.Store(), child.ParentState())
				if err != nil {
					return nil, nil, xerrors.Errorf("load state tree when considering message %s: %w", message.Cid().String(), err)
				}

				dstActorCode := accountActorCodeID
				dstActor, err := st.GetActor(dstAddr)
				if err != nil {
					// implicitly if actor does not exist,
					if !errors.Is(err, types.ErrActorNotFound) {
						return nil, nil, xerrors.Errorf("get destination actor for message %s failed: %w", message.Cid().String(), err)
					}
				} else {
					dstActorCode = dstActor.Code.String()
				}

				if pm, err := ParseMsg(message, ts, dstActorCode); err == nil {
					result.ParsedMessages = append(result.ParsedMessages, pm)
				} else {
					return nil, nil, xerrors.Errorf("parse message %s failed: %w", message.Cid().String(), err)
				}
			}
		}

	}
	newBaseFee := store.ComputeNextBaseFee(ts.Blocks()[0].ParentBaseFee, totalUniqGasLimit, len(ts.Blocks()), ts.Height())
	baseFeeRat := new(big.Rat).SetFrac(newBaseFee.Int, new(big.Int).SetUint64(build.FilecoinPrecision))
	baseFee, _ := baseFeeRat.Float64()

	baseFeeChange := new(big.Rat).SetFrac(newBaseFee.Int, ts.Blocks()[0].ParentBaseFee.Int)
	baseFeeChangeF, _ := baseFeeChange.Float64()

	result.MessageGasEconomy = &messagemodel.MessageGasEconomy{
		Height:              int64(ts.Height()),
		StateRoot:           ts.ParentState().String(),
		GasLimitTotal:       totalGasLimit,
		GasLimitUniqueTotal: totalUniqGasLimit,
		BaseFee:             baseFee,
		BaseFeeChangeLog:    math.Log(baseFeeChangeF) / math.Log(1.125),
		GasFillRatio:        float64(totalGasLimit) / float64(len(ts.Blocks())*build.BlockGasTarget),
		GasCapacityRatio:    float64(totalUniqGasLimit) / float64(len(ts.Blocks())*build.BlockGasTarget),
		GasWasteRatio:       float64(totalGasLimit-totalUniqGasLimit) / float64(len(ts.Blocks())*build.BlockGasTarget),
	}
	return result, pmsgModels, nil
}

func cidsEqual(c1, c2 []cid.Cid) bool {
	if len(c1) != len(c2) {
		return false
	}
	for i, c := range c1 {
		if !c2[i].Equals(c) {
			return false
		}
	}
	return true
}

// ParseMsg extracts message parameters and encodes them as JSON by looking at
// the messages destination actor type.
func ParseMsg(m *types.Message, ts *types.TipSet, destCode string) (*messagemodel.ParsedMessage, error) {
	pm := &messagemodel.ParsedMessage{
		Cid:    m.Cid().String(),
		Height: int64(ts.Height()),
		From:   m.From.String(),
		To:     m.To.String(),
		Value:  m.Value.String(),
	}

	actor, ok := statediff.LotusActorCodes[destCode]
	if !ok {
		actor = statediff.LotusTypeUnknown
	}
	var params ipld.Node
	var name string
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
		params, name, err = statediff.ParseParams(m.Params, int(m.Method), actor)
	}()
	if err != nil && actor != statediff.LotusTypeUnknown {
		// fall back to generic cbor->json conversion.
		actor = statediff.LotusTypeUnknown
		params, name, err = statediff.ParseParams(m.Params, int(m.Method), actor)
	}
	if name == "Unknown" {
		name = fmt.Sprintf("%s.%d", actor, m.Method)
	}
	pm.Method = name
	if err != nil {
		log.Warnf("failed to parse parameters of message %s: %v", m.Cid, err)
		// this can occur when the message is not valid cbor
		pm.Params = ""
		return pm, nil
	}
	if params != nil {
		buf := bytes.NewBuffer(nil)
		if err := fcjson.Encoder(params, buf); err != nil {
			return nil, xerrors.Errorf("json encode: %w", err)
		}
		pm.Params = string(bytes.ReplaceAll(bytes.ToValidUTF8(buf.Bytes(), []byte{}), []byte{0x00}, []byte{}))
	}

	return pm, nil
}

func (p *MessageProcessor) fetchReceipts(ctx context.Context, node lens.API, ts *types.TipSet) (messagemodel.Receipts, error) {
	out := messagemodel.Receipts{}

	// receipts and messages for a parent state are consistent for blocks in the same tipset.
	recs, err := node.ChainGetParentReceipts(ctx, ts.Blocks()[0].Cid())
	if err != nil {
		return nil, xerrors.Errorf("get parent receipts: %w", err)
	}
	msgs, err := node.ChainGetParentMessages(ctx, ts.Blocks()[0].Cid())
	if err != nil {
		return nil, xerrors.Errorf("get parent messages: %w", err)
	}

	for range ts.Cids() {
		// Stop processing if we have somehow passed our own lease time
		select {
		case <-ctx.Done():
			return nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		for i, r := range recs {
			out = append(out, &messagemodel.Receipt{
				Height:    int64(ts.Height()),
				Message:   msgs[i].Cid.String(),
				StateRoot: ts.ParentState().String(),
				Idx:       i,
				ExitCode:  int64(r.ExitCode),
				GasUsed:   r.GasUsed,
			})
		}
	}
	return out, nil
}

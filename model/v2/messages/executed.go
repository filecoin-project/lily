package messages

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

func init() {
	v2.RegisterExtractor(&ExecutedMessage{}, ExtractExecutedMessages)
}

var _ v2.LilyModel = (*ExecutedMessage)(nil)

// ExecutedMessage are message that were executed while applying a block.
type ExecutedMessage struct {
	Height      abi.ChainEpoch
	StateRoot   cid.Cid
	MessageCid  cid.Cid
	BlockCid    cid.Cid
	ToActorCode cid.Cid

	// Message
	From           address.Address
	To             address.Address
	Value          abi.TokenAmount
	GasFeeCap      abi.TokenAmount
	GasPremium     abi.TokenAmount
	SizeBytes      int64
	GasLimit       int64
	Nonce          uint64
	Method         abi.MethodNum
	MessageVersion uint64
	Params         []byte

	// Receipt
	ExitCode     exitcode.ExitCode
	ReceiptIndex int64
	GasUsed      int64
	Return       []byte

	// GasOutputs
	ParentBaseFee      abi.TokenAmount
	BaseFeeBurn        abi.TokenAmount
	OverEstimationBurn abi.TokenAmount
	MinerPenalty       abi.TokenAmount
	MinerTip           abi.TokenAmount
	Refund             abi.TokenAmount
	GasRefund          int64
	GasBurned          int64
}

func (t *ExecutedMessage) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(ExecutedMessage{}).Name()),
		Kind:    v2.ModelTsKind,
	}
}

func (t *ExecutedMessage) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func ExtractExecutedMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]v2.LilyModel, error) {
	grp, _ := errgroup.WithContext(ctx)

	var getActorCodeFn func(address address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCodeFn, err = util.MakeGetActorCodeFunc(ctx, api.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("getting actor code lookup function: %w", err)
		}
		return nil
	})

	var blkMsgRec []*lens.BlockMessageReceipts
	grp.Go(func() error {
		var err error
		blkMsgRec, err = api.TipSetMessageReceipts(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages and receipts: %w", err)
		}
		return nil
	})

	var burnFn lens.ShouldBurnFn
	grp.Go(func() error {
		var err error
		burnFn, err = api.ShouldBurnFn(ctx, executed)
		if err != nil {
			return fmt.Errorf("getting should burn function: %w", err)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	var out = make([]v2.LilyModel, 0, len(blkMsgRec))
	for _, msgrec := range blkMsgRec {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		itr, err := msgrec.Iterator()
		if err != nil {
			return nil, err
		}

		for itr.HasNext() {
			msg, recIdx, rec := itr.Next()

			toActorCode, found := getActorCodeFn(msg.VMMessage().To)
			if !found && rec.ExitCode == 0 {
				// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
				// message fails then Lotus will not allow the actor to be created and we are left with an address of an
				// unknown type.
				// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
				// indicates that Lily actor type detection is out of date.
				log.Errorw("parsing message", "cid", msg.Cid().String(), "receipt", rec)
				return nil, fmt.Errorf("failed to parse message params: missing to actor code")
			}

			gasOutputs, err := datasource.ComputeGasOutputs(ctx, msgrec.Block, msg.VMMessage(), rec, burnFn)
			if err != nil {
				return nil, fmt.Errorf("failed to compute gas outputs: %w", err)
			}

			out = append(out, &ExecutedMessage{
				Height:      msgrec.Block.Height,
				StateRoot:   msgrec.Block.ParentStateRoot,
				BlockCid:    msgrec.Block.ParentStateRoot,
				ToActorCode: toActorCode,

				MessageCid:     msg.Cid(),
				From:           msg.VMMessage().From,
				To:             msg.VMMessage().To,
				Value:          msg.VMMessage().Value,
				GasFeeCap:      msg.VMMessage().GasFeeCap,
				GasPremium:     msg.VMMessage().GasPremium,
				GasLimit:       msg.VMMessage().GasLimit,
				Nonce:          msg.VMMessage().Nonce,
				Method:         msg.VMMessage().Method,
				MessageVersion: msg.VMMessage().Version,
				Params:         msg.VMMessage().Params,
				SizeBytes:      int64(msg.ChainLength()),

				ReceiptIndex: int64(recIdx),
				ExitCode:     rec.ExitCode,
				GasUsed:      rec.GasUsed,
				Return:       rec.Return,

				ParentBaseFee:      msgrec.Block.ParentBaseFee,
				BaseFeeBurn:        gasOutputs.BaseFeeBurn,
				OverEstimationBurn: gasOutputs.OverEstimationBurn,
				MinerPenalty:       gasOutputs.MinerPenalty,
				MinerTip:           gasOutputs.MinerTip,
				Refund:             gasOutputs.Refund,
				GasRefund:          gasOutputs.GasRefund,
				GasBurned:          gasOutputs.GasBurned,
			})
		}
	}
	return out, nil
}

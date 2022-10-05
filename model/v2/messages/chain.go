package messages

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

func init() {
	v2.RegisterExtractor(&BlockMessage{}, ExtractBlockMessages)
}

var _ v2.LilyModel = (*BlockMessage)(nil)

// BlockMessage is a message on the chain which may or may not have been executed.
type BlockMessage struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	BlockCid  cid.Cid
	// Message
	MessageCid     cid.Cid
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
}

func (t *BlockMessage) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(BlockMessage{}).Name()),
		Kind:    v2.ModelTsKind,
	}
}

func (t *BlockMessage) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func (t *BlockMessage) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *BlockMessage) ToStorageBlock() (block.Block, error) {
	data, err := t.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (t *BlockMessage) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractBlockMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]v2.LilyModel, error) {
	blksMsgs, err := api.TipSetBlockMessages(ctx, current)
	if err != nil {
		return nil, err
	}

	var out = make([]v2.LilyModel, 0, len(blksMsgs))
	for _, blkMsgs := range blksMsgs {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}
		for _, msg := range blkMsgs.BlsMessages {
			out = append(out, &BlockMessage{
				Height:         blkMsgs.Block.Height,
				StateRoot:      blkMsgs.Block.ParentStateRoot,
				BlockCid:       blkMsgs.Block.Cid(),
				MessageCid:     msg.Cid(),
				From:           msg.From,
				To:             msg.To,
				Value:          msg.Value,
				GasFeeCap:      msg.GasFeeCap,
				GasPremium:     msg.GasPremium,
				SizeBytes:      int64(msg.ChainLength()),
				GasLimit:       msg.GasLimit,
				Nonce:          msg.Nonce,
				Method:         msg.Method,
				MessageVersion: msg.Version,
				Params:         msg.Params,
			})
		}
		for _, msg := range blkMsgs.SecpMessages {
			out = append(out, &BlockMessage{
				Height:         blkMsgs.Block.Height,
				StateRoot:      blkMsgs.Block.ParentStateRoot,
				BlockCid:       blkMsgs.Block.Cid(),
				MessageCid:     msg.Cid(),
				From:           msg.VMMessage().From,
				To:             msg.VMMessage().To,
				Value:          msg.VMMessage().Value,
				GasFeeCap:      msg.VMMessage().GasFeeCap,
				GasPremium:     msg.VMMessage().GasPremium,
				SizeBytes:      int64(msg.ChainLength()),
				GasLimit:       msg.VMMessage().GasLimit,
				Nonce:          msg.VMMessage().Nonce,
				Method:         msg.VMMessage().Method,
				MessageVersion: msg.VMMessage().Version,
				Params:         msg.VMMessage().Params,
			})
		}
	}
	return out, nil
}

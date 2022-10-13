package economics

import (
	"bytes"
	"context"
	"reflect"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

func init() {
	v2.RegisterExtractor(&ChainEconomics{}, Extract)
}

type ChainEconomics struct {
	Height               abi.ChainEpoch
	StateRoot            cid.Cid
	TotalGasLimit        int64
	TotalUniqueGasLimit  int64
	NumBlocks            int64
	ParentBaseFee        abi.TokenAmount
	BaseFee              abi.TokenAmount
	FilVested            abi.TokenAmount
	FilMined             abi.TokenAmount
	FilBurnt             abi.TokenAmount
	FilLocked            abi.TokenAmount
	FilCirculating       abi.TokenAmount
	FilReservedDisbursed abi.TokenAmount
}

func (m *ChainEconomics) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(ChainEconomics{}).Name()),
		Kind:    v2.ModelTsKind,
	}
}
func (t *ChainEconomics) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func (t *ChainEconomics) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *ChainEconomics) ToStorageBlock() (block.Block, error) {
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

func (t *ChainEconomics) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func Extract(ctx context.Context, api tasks.DataSource, current *types.TipSet, executed *types.TipSet) ([]v2.LilyModel, error) {
	blksMsgs, err := api.TipSetBlockMessages(ctx, current)
	if err != nil {
		return nil, err
	}
	var (
		seenMsgs          = cid.NewSet()
		totalGasLimit     int64
		totalUniqGasLimit int64
		baseFee           abi.TokenAmount
	)
	for _, msgs := range blksMsgs {
		for _, msg := range msgs.SecpMessages {
			totalGasLimit += msg.Message.GasLimit
			if seenMsgs.Visit(msg.Cid()) {
				totalUniqGasLimit += msg.Message.GasLimit
			}
		}
		for _, msg := range msgs.BlsMessages {
			totalGasLimit += msg.GasLimit
			if seenMsgs.Visit(msg.Cid()) {
				totalUniqGasLimit += msg.GasLimit
			}
		}
	}
	baseFee, err = api.ComputeBaseFee(ctx, current)
	if err != nil {
		return nil, err
	}
	supply, err := api.CirculatingSupply(ctx, current)
	if err != nil {
		return nil, err
	}

	return []v2.LilyModel{
		&ChainEconomics{
			Height:               current.Height(),
			StateRoot:            current.ParentState(),
			TotalGasLimit:        totalGasLimit,
			TotalUniqueGasLimit:  totalUniqGasLimit,
			NumBlocks:            int64(len(current.Blocks())),
			ParentBaseFee:        current.Blocks()[0].ParentBaseFee,
			BaseFee:              baseFee,
			FilVested:            supply.FilVested,
			FilMined:             supply.FilMined,
			FilBurnt:             supply.FilBurnt,
			FilLocked:            supply.FilLocked,
			FilCirculating:       supply.FilCirculating,
			FilReservedDisbursed: supply.FilReserveDisbursed,
		},
	}, nil
}

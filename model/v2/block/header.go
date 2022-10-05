package block

import (
	"bytes"
	"context"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/proof"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

func init() {
	v2.RegisterExtractor(&BlockHeader{}, Extract)
}

var _ v2.LilyModel = (*BlockHeader)(nil)

type BlockHeader struct {
	Height                abi.ChainEpoch
	StateRoot             cid.Cid
	BlockCID              cid.Cid
	Miner                 address.Address
	Ticket                *types.Ticket
	ElectionProof         *types.ElectionProof
	BeaconEntries         []types.BeaconEntry
	WinPoStProof          []proof.PoStProof
	Parents               []cid.Cid
	ParentWeight          types.BigInt
	ParentMessageReceipts cid.Cid
	Messages              cid.Cid
	BLSAggregate          *crypto.Signature
	Timestamp             uint64
	BlockSig              *crypto.Signature
	ForkSignaling         uint64
	ParentBaseFee         abi.TokenAmount
}

func (b *BlockHeader) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Type:    v2.ModelType(reflect.TypeOf(BlockHeader{}).Name()),
		Kind:    v2.ModelTsKind,
		Version: 1,
	}
}

func (b *BlockHeader) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    b.Height,
		StateRoot: b.StateRoot,
	}
}

func (t *BlockHeader) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *BlockHeader) ToStorageBlock() (block.Block, error) {
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

func (t *BlockHeader) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func Extract(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]v2.LilyModel, error) {
	out := make([]v2.LilyModel, len(current.Blocks()))
	for i, bh := range current.Blocks() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			out[i] = &BlockHeader{
				Height:                bh.Height,
				StateRoot:             bh.ParentStateRoot,
				BlockCID:              bh.Cid(),
				Miner:                 bh.Miner,
				Ticket:                bh.Ticket,
				ElectionProof:         bh.ElectionProof,
				BeaconEntries:         bh.BeaconEntries,
				WinPoStProof:          bh.WinPoStProof,
				Parents:               bh.Parents,
				ParentWeight:          bh.ParentWeight,
				ParentMessageReceipts: bh.ParentMessageReceipts,
				Messages:              bh.Messages,
				BLSAggregate:          bh.BLSAggregate,
				Timestamp:             bh.Timestamp,
				BlockSig:              bh.BlockSig,
				ForkSignaling:         bh.ForkSignaling,
				ParentBaseFee:         bh.ParentBaseFee,
			}
		}
	}
	return out, nil
}

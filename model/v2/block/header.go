package block

import (
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/proof"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	v2 "github.com/filecoin-project/lily/model/v2"
)

type BlockHeader struct {
	Height                abi.ChainEpoch
	StateRoot             cid.Cid
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

func (b *BlockHeader) Type() v2.ModelType {
	// eww gross
	return v2.ModelType(reflect.TypeOf(BlockHeader{}).Name())
}

func (b *BlockHeader) Version() v2.ModelVersion {
	return 1
}

func (b *BlockHeader) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    b.Height,
		StateRoot: b.StateRoot,
	}
}

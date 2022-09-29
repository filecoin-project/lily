package messages

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
)

type Message struct {
	Height         abi.ChainEpoch
	StateRoot      cid.Cid
	MessageCid     cid.Cid
	ToActorCode    cid.Cid
	From           address.Address
	To             address.Address
	Value          abi.TokenAmount
	GasFeeCap      abi.TokenAmount
	GasPremium     abi.TokenAmount
	SizeBytes      int64
	GasLimit       int64
	Nonce          uint64
	Method         uint64
	MessageVersion uint64
	Params         []byte
}

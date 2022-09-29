package messages

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"
)

type Receipt struct {
	Height             abi.ChainEpoch
	StateRoot          cid.Cid
	MessageCid         cid.Cid
	MessageToActorCode cid.Cid
	ExitCode           exitcode.ExitCode
	Index              int64
	GasUsed            int64
	Return             []byte
}

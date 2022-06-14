package messages

import (
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/lens"
)

type MessageError struct {
	Cid   cid.Cid
	Error string
}

func walkExecutionTrace(et *types.ExecutionTrace, trace *[]*MessageTrace) {
	for _, sub := range et.Subcalls {
		*trace = append(*trace, &MessageTrace{
			Message:   sub.Msg,
			Receipt:   sub.MsgRct,
			Error:     sub.Error,
			Duration:  sub.Duration,
			GasCharge: sub.GasCharges,
		})
		walkExecutionTrace(&sub, trace) //nolint:scopelint,gosec
	}
}

type MessageTrace struct {
	Message   *types.Message
	Receipt   *types.MessageReceipt
	Error     string
	Duration  time.Duration
	GasCharge []*types.GasTrace
}

func GetChildMessagesOf(m *lens.MessageExecution) []*MessageTrace {
	var out []*MessageTrace
	walkExecutionTrace(&m.Ret.ExecutionTrace, &out)
	return out
}

package processor

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/lens"
)

type VmMessage struct {
	Source  cid.Cid              `cborgen:"source"`
	Message *types.Message       `cborgen:"message"`
	Receipt types.MessageReceipt `cborgen:"receipt"`
	// TODO these traces can become very long (over 200,000 entires) meaning its a bad idea to encode this as an array in cbor (it will break)
	// if there is value in gathering this information then we'll want to put it in an AMT.
	//GasTrace []*VmMessageGasTrace `cborgen:"gas"`
	Error string `cborgen:"error"`
	Index int64  `cborgen:"index"`
}

type VmMessageGasTrace struct {
	Name              string `cborgen:"name"`
	Location          []Loc  `cborgen:"location"`
	TotalGas          int64  `cborgen:"totalgas"`
	ComputeGas        int64  `cborgen:"computegas"`
	StorageGas        int64  `cborgen:"storagegas"`
	TotalVirtualGas   int64  `cborgen:"totalvirtgas"`
	VirtualComputeGas int64  `cborgen:"virtcomputegas"`
	VirtualStorageGas int64  `cborgen:"cirtstoragegas"`
}

type Loc struct {
	File     string `cborgen:"file"`
	Line     int64  `cborgen:"line"`
	Function string `cborgen:"function"`
}

type VmMessageList []*VmMessage

func (vml VmMessageList) ToAdtArray(store adt.Store, bw int) (cid.Cid, error) {
	msgAmt, err := adt.MakeEmptyArray(store, bw)
	if err != nil {
		return cid.Undef, nil
	}

	for _, msg := range vml {
		if err := msgAmt.Set(uint64(msg.Index), msg); err != nil {
			return cid.Undef, err
		}
	}
	return msgAmt.Root()
}

func VmMessageListFromAdtArray(store adt.Store, root cid.Cid, bw int) (VmMessageList, error) {
	arr, err := adt.AsArray(store, root, bw)
	if err != nil {
		return nil, err
	}
	out := make(VmMessageList, arr.Length())
	msg := new(VmMessage)
	idx := 0
	if err := arr.ForEach(msg, func(i int64) error {
		val := new(VmMessage)
		*val = *msg
		out[idx] = val
		idx++
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func ProcessVmMessages(ctx context.Context, source *lens.MessageExecutionV2) (VmMessageList, error) {
	trace := GetChildMessagesOf(source)
	out := make([]*VmMessage, len(trace))
	for traceIdx, vmmsg := range trace {
		// see TODO on VmMessage struct
		/*
			vmGas := make([]*VmMessageGasTrace, len(vmmsg.GasCharge))

			for gasIdx, g := range vmmsg.GasCharge {
				loc := make([]Loc, len(g.Location))

				for locIdx, l := range g.Location {
					loc[locIdx] = Loc{
						File:     l.File,
						Line:     int64(l.Line),
						Function: l.Function,
					}
				}

				vmGas[gasIdx] = &VmMessageGasTrace{
					Name:              g.Name,
					Location:          loc,
					TotalGas:          g.TotalGas,
					ComputeGas:        g.ComputeGas,
					StorageGas:        g.StorageGas,
					TotalVirtualGas:   g.TotalVirtualGas,
					VirtualComputeGas: g.VirtualComputeGas,
					VirtualStorageGas: g.VirtualStorageGas,
				}
			}

		*/

		out[traceIdx] = &VmMessage{
			Source:  source.Cid,
			Message: vmmsg.Message,
			Receipt: *vmmsg.Receipt,
			//GasTrace: vmGas,
			Error: vmmsg.Error,
			Index: int64(vmmsg.Index),
		}
	}
	return out, nil
}

type vmMessageTrace struct {
	Message   *types.Message
	Receipt   *types.MessageReceipt
	Error     string
	Duration  time.Duration
	GasCharge []*types.GasTrace
	Index     int
}

func GetChildMessagesOf(m *lens.MessageExecutionV2) []*vmMessageTrace {
	var out []*vmMessageTrace
	index := 0
	walkExecutionTrace(&m.Ret.ExecutionTrace, &out, &index)
	return out
}

func walkExecutionTrace(et *types.ExecutionTrace, trace *[]*vmMessageTrace, index *int) {
	for _, sub := range et.Subcalls {
		*trace = append(*trace, &vmMessageTrace{
			Message:   sub.Msg,
			Receipt:   sub.MsgRct,
			Error:     sub.Error,
			Duration:  sub.Duration,
			GasCharge: sub.GasCharges,
			Index:     *index,
		})
		*index++
		walkExecutionTrace(&sub, trace, index) //nolint:scopelint,gosec
	}
}

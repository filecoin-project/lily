package core

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/tasks"
)

type VMMessage struct {
	SourceCID cid.Cid
	Message   *types.Message
	Receipt   *types.MessageReceipt
	Error     string
	GasTrace  []*GasTrace
}

type GasTrace struct {
	Name              string
	TotalGas          int64
	ComputeGas        int64
	StorageGas        int64
	TotalVirtualGas   int64
	VirtualComputeGas int64
	VirtualStorageGas int64
}

func GasTraceFor(m *util.MessageTrace) []*GasTrace {
	out := make([]*GasTrace, len(m.GasCharge))
	for i, g := range m.GasCharge {
		out[i] = &GasTrace{
			Name:              g.Name,
			TotalGas:          g.TotalGas,
			ComputeGas:        g.ComputeGas,
			StorageGas:        g.StorageGas,
			TotalVirtualGas:   g.TotalVirtualGas,
			VirtualComputeGas: g.VirtualComputeGas,
			VirtualStorageGas: g.VirtualStorageGas,
		}
	}
	return out
}

func ExtractVMMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]*VMMessage, error) {
	mex, err := api.MessageExecutions(ctx, current, executed)
	if err != nil {
		return nil, fmt.Errorf("getting messages executions for tipset: %w", err)
	}

	out := make([]*VMMessage, 0, len(mex))
	for _, parentMsg := range mex {
		for _, child := range util.GetChildMessagesOf(parentMsg) {
			parentMsg := parentMsg
			out = append(out, &VMMessage{
				SourceCID: parentMsg.Cid,
				Message:   child.Message,
				Receipt:   child.Receipt,
				Error:     child.Error,
				GasTrace:  GasTraceFor(child),
			})
		}
	}
	return out, nil
}

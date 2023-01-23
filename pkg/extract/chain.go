package extract

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/chain"
	"github.com/filecoin-project/lily/tasks"
)

type MessageStateChanges struct {
	Current          *types.TipSet
	Executed         *types.TipSet
	BaseFee          abi.TokenAmount
	FullBlocks       map[cid.Cid]*chain.FullBlock
	ImplicitMessages []*chain.ImplicitMessage
}

func FullBlockMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*MessageStateChanges, error) {
	baseFee, err := api.ComputeBaseFee(ctx, executed)
	if err != nil {
		return nil, err
	}
	fullBlocks, implicitMessages, err := chain.Messages(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}

	return &MessageStateChanges{
		Current:          current,
		Executed:         executed,
		BaseFee:          baseFee,
		FullBlocks:       fullBlocks,
		ImplicitMessages: implicitMessages,
	}, nil
}

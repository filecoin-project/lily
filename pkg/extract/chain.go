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
	BaseFee           abi.TokenAmount
	CirculatingSupply CirculatingSupply
	FullBlocks        map[cid.Cid]*chain.FullBlock
	ImplicitMessages  []*chain.ImplicitMessage
}

type CirculatingSupply struct {
	FilVested           abi.TokenAmount
	FilMined            abi.TokenAmount
	FilBurnt            abi.TokenAmount
	FilLocked           abi.TokenAmount
	FilCirculating      abi.TokenAmount
	FilReserveDisbursed abi.TokenAmount
}

func FullBlockMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*MessageStateChanges, error) {
	baseFee, err := api.ComputeBaseFee(ctx, executed)
	if err != nil {
		return nil, err
	}
	cs, err := api.CirculatingSupply(ctx, current)
	if err != nil {
		return nil, err
	}
	fullBlocks, implicitMessages, err := chain.Messages(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}

	return &MessageStateChanges{
		BaseFee: baseFee,
		CirculatingSupply: CirculatingSupply{
			FilVested:           cs.FilVested,
			FilMined:            cs.FilMined,
			FilBurnt:            cs.FilBurnt,
			FilLocked:           cs.FilLocked,
			FilCirculating:      cs.FilCirculating,
			FilReserveDisbursed: cs.FilReserveDisbursed,
		},
		FullBlocks:       fullBlocks,
		ImplicitMessages: implicitMessages,
	}, nil
}

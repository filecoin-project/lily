package core

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/tasks"
)

type Supply struct {
	BaseFee              abi.TokenAmount
	FilVested            abi.TokenAmount
	FilMined             abi.TokenAmount
	FilBurnt             abi.TokenAmount
	FilLocked            abi.TokenAmount
	FilCirculating       abi.TokenAmount
	FilReservedDisbursed abi.TokenAmount
}

func ExtractTokenSupply(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*Supply, error) {
	baseFee, err := api.ComputeBaseFee(ctx, current)
	if err != nil {
		return nil, err
	}
	circulating, err := api.CirculatingSupply(ctx, current)
	if err != nil {
		return nil, err
	}

	return &Supply{
		BaseFee:              baseFee,
		FilVested:            circulating.FilVested,
		FilMined:             circulating.FilMined,
		FilBurnt:             circulating.FilBurnt,
		FilLocked:            circulating.FilLocked,
		FilCirculating:       circulating.FilCirculating,
		FilReservedDisbursed: circulating.FilReserveDisbursed,
	}, nil
}

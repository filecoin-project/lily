package lens

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
)

type API interface {
	Store() adt.Store
	api.FullNode
	ComputeGasOutputs(gasUsed, gasLimit int64, baseFee, feeCap, gasPremium abi.TokenAmount) vm.GasOutputs
}

type APICloser func()

type APIOpener interface {
	Open(context.Context) (API, APICloser, error)
}

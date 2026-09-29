package reward

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	reward19 "github.com/filecoin-project/go-state-types/builtin/v19/reward"
)

func TestComputeStreamWeight(t *testing.T) {
	const denom = reward19.Denom
	w1Start := denom / 100 * 95
	w2Start := denom / 100 * 5

	consensus := reward19.WeightRecord{
		VStart: w1Start,
		Slope:  -1,
		TStart: 100,
		Floor:  denom / 100 * 50,
		Cap:    w1Start,
	}
	service := reward19.WeightRecord{
		VStart: w2Start,
		Slope:  1,
		TStart: 100,
		Floor:  w2Start,
		Cap:    denom / 100 * 10,
	}

	atStart := abi.ChainEpoch(100)
	require.True(t, ComputeStreamWeight(consensus, atStart).Equals(big.NewIntUnsigned(w1Start)))
	require.True(t, ComputeStreamWeight(service, atStart).Equals(big.NewIntUnsigned(w2Start)))
	require.True(t, BurnWeight([]big.Int{
		ComputeStreamWeight(consensus, atStart),
		ComputeStreamWeight(service, atStart),
	}).Equals(big.Zero()))

	next := abi.ChainEpoch(101)
	require.True(t, ComputeStreamWeight(consensus, next).Equals(big.NewIntUnsigned(w1Start-1)))
	require.True(t, ComputeStreamWeight(service, next).Equals(big.NewIntUnsigned(w2Start+1)))
	require.True(t, BurnWeight([]big.Int{
		ComputeStreamWeight(consensus, next),
		ComputeStreamWeight(service, next),
	}).Equals(big.Zero()))

	// A record above its cap clamps, including when evaluated before t_start.
	clamped := reward19.WeightRecord{VStart: 150, Slope: -10, TStart: 100, Floor: 0, Cap: 100}
	require.True(t, ComputeStreamWeight(clamped, 90).Equals(big.NewIntUnsigned(100)))
	require.True(t, ComputeStreamWeight(clamped, 200).Equals(big.Zero()))
}

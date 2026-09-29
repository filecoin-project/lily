package reward

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	reward19 "github.com/filecoin-project/go-state-types/builtin/v19/reward"
	adt19 "github.com/filecoin-project/go-state-types/builtin/v19/util/adt"
)

// Stream ids pinned by the FIP-0118 activation migration.
const (
	ConsensusStreamID reward19.StreamID = 1
	ServiceStreamID   reward19.StreamID = 2
)

// StreamLedger is the v19 reward-actor view of block-reward streams.
// Earlier reward actor versions do not implement it.
type StreamLedger interface {
	TotalMintedReward() abi.TokenAmount
	TotalBurnMinted() abi.TokenAmount
	TotalExplicitMinted() abi.TokenAmount
	LoadStreams() (*reward19.StreamsState, error)
}

func (s *state19) TotalMintedReward() abi.TokenAmount {
	return s.State.TotalMintedReward
}

func (s *state19) TotalBurnMinted() abi.TokenAmount {
	return s.State.TotalBurnMinted
}

func (s *state19) TotalExplicitMinted() abi.TokenAmount {
	return s.State.TotalExplicitMinted
}

func (s *state19) LoadStreams() (*reward19.StreamsState, error) {
	return s.State.LoadStreams(adt19.WrapStore(s.store.Context(), s.store))
}

// ComputeStreamWeight evaluates one clamped linear weight at epoch.
// v_start, floor, and cap are DENOM fixed point; slope is signed DENOM fixed point per epoch.
func ComputeStreamWeight(w reward19.WeightRecord, epoch abi.ChainEpoch) big.Int {
	delta := int64(epoch - w.TStart)
	v := big.Add(big.NewIntUnsigned(w.VStart), big.Mul(big.NewInt(w.Slope), big.NewInt(delta)))
	floor := big.NewIntUnsigned(w.Floor)
	cap := big.NewIntUnsigned(w.Cap)
	if v.LessThan(floor) {
		return floor
	}
	if v.GreaterThan(cap) {
		return cap
	}
	return v
}

// BurnWeight is the residual DENOM - sum(evaluated stream weights), i.e. w0 = DENOM - (w1 + w2 + ...).
// It counts every live stream, including any stream other than consensus and service, so w0+w1+w2 always sums to DENOM (1e18).
func BurnWeight(evaluated []big.Int) big.Int {
	sum := big.Zero()
	for _, w := range evaluated {
		sum = big.Add(sum, w)
	}
	return big.Sub(big.NewIntUnsigned(reward19.Denom), sum)
}

package v6

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin/power"

	"github.com/filecoin-project/lily/model"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
)

func ExtractPowerStateChanges(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *rawdiff.ActorChange) (model.Persistable, error) {
	if change.Change == core.ChangeTypeRemove {
		panic("power actor should never be removed from the state tree")
	}
	powerState := new(power.State)
	if err := powerState.UnmarshalCBOR(bytes.NewReader(change.Current)); err != nil {
		return nil, err
	}
	return &powermodel.ChainPower{
		Height:                     int64(current.Height()),
		StateRoot:                  current.ParentState().String(),
		TotalRawBytesPower:         powerState.TotalRawBytePower.String(),
		TotalQABytesPower:          powerState.TotalQualityAdjPower.String(),
		TotalRawBytesCommitted:     powerState.TotalBytesCommitted.String(),
		TotalQABytesCommitted:      powerState.TotalQABytesCommitted.String(),
		TotalPledgeCollateral:      powerState.TotalPledgeCollateral.String(),
		QASmoothedPositionEstimate: powerState.ThisEpochQAPowerSmoothed.PositionEstimate.String(),
		QASmoothedVelocityEstimate: powerState.ThisEpochQAPowerSmoothed.VelocityEstimate.String(),
		MinerCount:                 uint64(powerState.MinerCount),
		ParticipatingMinerCount:    uint64(powerState.MinerAboveMinPowerCount),
	}, nil
}

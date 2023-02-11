package v8

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/builtin/v8/power"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	"github.com/filecoin-project/lily/pkg/core"
	powerdiff "github.com/filecoin-project/lily/pkg/extract/actors/powerdiff/v1"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

type Claims struct{}

func (Claims) Transform(ctx context.Context, current, executed *types.TipSet, changes *powerdiff.StateDiffResult) model.Persistable {
	report := data.StartProcessingReport(tasktype.PowerActorClaim, current)
	for _, change := range changes.ClaimsChanges {
		// only care about new and modified power entries
		if change.Change == core.ChangeTypeRemove {
			continue
		}
		miner, err := address.NewFromBytes(change.Miner)
		if err != nil {
			report.AddError(err)
			continue
		}
		claim := new(power.Claim)
		if err := claim.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
			report.AddError(err)
			continue
		}
		report.AddModels(&powermodel.PowerActorClaim{
			Height:          int64(current.Height()),
			MinerID:         miner.String(),
			StateRoot:       current.ParentState().String(),
			RawBytePower:    claim.RawBytePower.String(),
			QualityAdjPower: claim.QualityAdjPower.String(),
		})
	}
	return report.Finish()
}

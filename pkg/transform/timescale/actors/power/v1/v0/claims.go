package v0

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"

	"github.com/filecoin-project/lily/model"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	"github.com/filecoin-project/lily/pkg/core"
	powerdiff "github.com/filecoin-project/lily/pkg/extract/actors/powerdiff/v0"
)

type Claims struct{}

func (Claims) Transform(ctx context.Context, current, executed *types.TipSet, changes *powerdiff.StateDiffResult) (model.Persistable, error) {
	out := make(powermodel.PowerActorClaimList, 0, len(changes.ClaimsChanges))
	for _, change := range changes.ClaimsChanges {
		// only care about new and modified power entries
		if change.Change == core.ChangeTypeRemove {
			continue
		}
		miner, err := address.NewFromBytes(change.Miner)
		if err != nil {
			return nil, err
		}
		claim := new(power.Claim)
		if err := claim.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
			return nil, err
		}
		out = append(out, &powermodel.PowerActorClaim{
			Height:          int64(current.Height()),
			MinerID:         miner.String(),
			StateRoot:       current.ParentState().String(),
			RawBytePower:    claim.RawBytePower.String(),
			QualityAdjPower: claim.QualityAdjPower.String(),
		})
	}
	return out, nil
}

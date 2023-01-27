package init

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	initmodel "github.com/filecoin-project/lily/model/actors/init"
	"github.com/filecoin-project/lily/pkg/core"
	initdiff "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v1"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

type Addresses struct{}

func (a Addresses) Extract(ctx context.Context, current, executed *types.TipSet, changes *initdiff.StateDiffResult) model.Persistable {
	report := data.StartProcessingReport(tasktype.IDAddress, current)
	for _, change := range changes.AddressesChanges {
		if change.Change == core.ChangeTypeAdd || change.Change == core.ChangeTypeModify {
			robustAddr, err := address.NewFromBytes(change.Address)
			if err != nil {
				report.AddError(err)
				continue
			}
			var actorID cbg.CborInt
			if err := actorID.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
				report.AddError(err)
				continue
			}
			idAddr, err := address.NewIDAddress(uint64(actorID))
			if err != nil {
				report.AddError(err)
				continue
			}
			report.AddModels(&initmodel.IDAddress{
				Height:    int64(current.Height()),
				StateRoot: current.ParentState().String(),
				ID:        idAddr.String(),
				Address:   robustAddr.String(),
			})
		}
	}
	return report.Finish()
}

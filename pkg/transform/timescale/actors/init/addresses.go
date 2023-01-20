package init

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/model"
	initmodel "github.com/filecoin-project/lily/model/actors/init"
	"github.com/filecoin-project/lily/pkg/core"
	initdiff "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v0"
)

type Addresses struct{}

func (a Addresses) Extract(ctx context.Context, current, executed *types.TipSet, changes *initdiff.StateDiffResult) (model.Persistable, error) {
	out := initmodel.IDAddressList{}
	for _, change := range changes.AddressesChanges {
		if change.Change == core.ChangeTypeAdd || change.Change == core.ChangeTypeModify {
			robustAddr, err := address.NewFromBytes(change.Address)
			if err != nil {
				return nil, err
			}
			var actorID cbg.CborInt
			if err := actorID.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
				return nil, err
			}
			idAddr, err := address.NewIDAddress(uint64(actorID))
			if err != nil {
				return nil, err
			}
			out = append(out, &initmodel.IDAddress{
				Height:    int64(current.Height()),
				StateRoot: current.ParentState().String(),
				ID:        idAddr.String(),
				Address:   robustAddr.String(),
			})
		}
	}
	return out, nil
}

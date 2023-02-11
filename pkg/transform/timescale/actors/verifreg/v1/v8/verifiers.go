package v8

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/builtin/v8/verifreg"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	"github.com/filecoin-project/lily/pkg/core"
	verifregdiff "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v1"
)

type Verifiers struct{}

func (Verifiers) Transform(ctx context.Context, current, executed *types.TipSet, changes *verifregdiff.StateDiffResult) (model.Persistable, error) {
	out := make(verifregmodel.VerifiedRegistryVerifiersList, len(changes.VerifierChanges))
	for i, change := range changes.VerifierChanges {
		addr, err := address.NewFromBytes(change.Verifier)
		if err != nil {
			return nil, err
		}
		switch change.Change {
		case core.ChangeTypeRemove:
			out[i] = &verifregmodel.VerifiedRegistryVerifier{
				Height:    int64(current.Height()),
				StateRoot: current.ParentState().String(),
				Address:   addr.String(),
				Event:     verifregmodel.Removed,
				DataCap:   "0", // data cap remove is zero
			}
		case core.ChangeTypeAdd:
			dcap := new(verifreg.DataCap)
			if err := dcap.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
				return nil, err
			}
			out[i] = &verifregmodel.VerifiedRegistryVerifier{
				Height:    int64(current.Height()),
				StateRoot: current.ParentState().String(),
				Address:   addr.String(),
				Event:     verifregmodel.Added,
				DataCap:   dcap.String(),
			}
		case core.ChangeTypeModify:
			dcap := new(verifreg.DataCap)
			if err := dcap.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
				return nil, err
			}
			out[i] = &verifregmodel.VerifiedRegistryVerifier{
				Height:    int64(current.Height()),
				StateRoot: current.ParentState().String(),
				Address:   addr.String(),
				Event:     verifregmodel.Modified,
				DataCap:   dcap.String(),
			}
		}
	}
	return out, nil
}

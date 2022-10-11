package verifreg

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/verifreg"
	"github.com/filecoin-project/lily/tasks"
)

type VerifierTransform struct {
	meta v2.ModelMeta
}

func NewVerifierTransform() *VerifierTransform {
	info := verifreg.Verifier{}
	return &VerifierTransform{meta: info.Meta()}
}

func (s *VerifierTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(verifregmodel.VerifiedRegistryVerifiersList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				vv := modeldata.(*verifreg.Verifier)
				sqlModels = append(sqlModels, &verifregmodel.VerifiedRegistryVerifier{
					Height:    int64(vv.Height),
					StateRoot: vv.StateRoot.String(),
					Address:   vv.Verifier.String(),
					Event:     vv.Event.String(),
					DataCap:   vv.DataCap.String(),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *VerifierTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *VerifierTransform) Name() string {
	info := VerifierTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *VerifierTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

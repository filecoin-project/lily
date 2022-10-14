package verifreg

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/verifreg"
)

type VerifierTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewVerifierTransform(taskName string) *VerifierTransform {
	info := verifreg.Verifier{}
	return &VerifierTransform{meta: info.Meta(), taskName: taskName}
}

func (s *VerifierTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(verifregmodel.VerifiedRegistryVerifiersList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
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
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
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

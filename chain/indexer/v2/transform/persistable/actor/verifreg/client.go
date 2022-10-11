package verifreg

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/verifreg"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("transform/verifreg")

type VerifiedClientTransform struct {
	meta v2.ModelMeta
}

func NewVerifiedClientTransform() *VerifiedClientTransform {
	info := verifreg.VerifiedClient{}
	return &VerifiedClientTransform{meta: info.Meta()}
}

func (s *VerifiedClientTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(verifregmodel.VerifiedRegistryVerifiedClientsList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				vc := modeldata.(*verifreg.VerifiedClient)
				sqlModels = append(sqlModels, &verifregmodel.VerifiedRegistryVerifiedClient{
					Height:    int64(vc.Height),
					StateRoot: vc.StateRoot.String(),
					Address:   vc.Client.String(),
					Event:     vc.Event.String(),
					DataCap:   vc.DataCap.String(),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *VerifiedClientTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *VerifiedClientTransform) Name() string {
	info := VerifiedClientTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *VerifiedClientTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

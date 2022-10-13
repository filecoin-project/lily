package cborable

import (
	"context"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	v2 "github.com/filecoin-project/lily/model/v2"
)

type CborTransform struct {
	meta v2.ModelMeta
}

func NewCborTransform() *CborTransform {
	return &CborTransform{}
}

func (c *CborTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			models := make([]v2.LilyModel, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				models = append(models, modeldata)
			}
			if len(models) > 0 {
				out <- &Result{Model: models, TipSet: res.Current()}
			}
		}
	}
	return nil
}

func (c *CborTransform) Name() string {
	return reflect.TypeOf(CborTransform{}).Name()
}

func (c *CborTransform) ModelType() v2.ModelMeta {
	return v2.ModelMeta{}
}

func (c *CborTransform) Matcher() string {
	return ".*"
}

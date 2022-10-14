package cborable

import (
	"context"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	v2 "github.com/filecoin-project/lily/model/v2"
)

type CborTipSetTransform struct {
	meta v2.ModelMeta
}

func NewCborTipSetTransform() *CborTipSetTransform {
	return &CborTipSetTransform{}
}

func (c *CborTipSetTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			models := make([]v2.LilyModel, 0, len(res.Models))
			for _, modeldata := range res.Models {
				models = append(models, modeldata)
			}
			if len(models) > 0 {
				out <- &Result{Model: models, TipSet: res.TipSet}
			}
		}
	}
	return nil
}
func (c *CborTipSetTransform) Name() string {
	return reflect.TypeOf(CborTipSetTransform{}).Name()
}

func (c *CborTipSetTransform) ModelType() v2.ModelMeta {
	return v2.ModelMeta{}
}

func (c *CborTipSetTransform) Matcher() string {
	return ".*"
}

type CborActorTransform struct {
	meta v2.ModelMeta
}

func NewCborActorTransform() *CborActorTransform {
	return &CborActorTransform{}
}

func (c *CborActorTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			models := make([]v2.LilyModel, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				models = append(models, modeldata)
			}
			if len(models) > 0 {
				out <- &Result{Model: models, TipSet: res.TipSet}
			}
		}
	}
	return nil
}
func (c *CborActorTransform) Name() string {
	return reflect.TypeOf(CborActorTransform{}).Name()
}

func (c *CborActorTransform) ModelType() v2.ModelMeta {
	return v2.ModelMeta{}
}

func (c *CborActorTransform) Matcher() string {
	return ".*"
}

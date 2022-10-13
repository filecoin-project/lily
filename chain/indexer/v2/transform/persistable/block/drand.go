package block

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model/blocks"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/block"
)

type DrandBlockEntryTransform struct {
	meta v2.ModelMeta
}

func NewDrandBlockEntryTransform() *DrandBlockEntryTransform {
	info := block.BlockHeader{}
	return &DrandBlockEntryTransform{meta: info.Meta()}
}

func (b *DrandBlockEntryTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(blocks.DrandBlockEntries, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
				m := modeldata.(*block.BlockHeader)
				for _, ent := range m.BeaconEntries {
					sqlModels = append(sqlModels, &blocks.DrandBlockEntrie{
						Round: ent.Round,
						Block: m.BlockCID.String(),
					})

				}
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (b *DrandBlockEntryTransform) ModelType() v2.ModelMeta {
	return b.meta
}

func (b *DrandBlockEntryTransform) Name() string {
	info := DrandBlockEntryTransform{}
	return reflect.TypeOf(info).Name()
}

func (b *DrandBlockEntryTransform) Matcher() string {
	return fmt.Sprintf("^%s$", b.meta.String())
}

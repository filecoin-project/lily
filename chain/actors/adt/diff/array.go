package diff

import (
	"context"

	"github.com/filecoin-project/go-amt-ipld/v3"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	adt2 "github.com/filecoin-project/sentinel-visor/chain/actors/adt"
)

func Amt(ctx context.Context, preArr, curArr adt2.Array, preStore, curStore adt.Store, amtOpts ...amt.Option) ([]*amt.Change, error) {
	preRoot, err := preArr.Root()
	if err != nil {
		return nil, err
	}

	curRoot, err := curArr.Root()
	if err != nil {
		return nil, err
	}

	return amt.Diff(ctx, preStore, curStore, preRoot, curRoot, amtOpts...)
}

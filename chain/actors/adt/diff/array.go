package diff

import (
	"context"

	"github.com/filecoin-project/go-amt-ipld/v3"
	adt2 "github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/adt"
)

// Amt returns a set of changes that transform `preArr` into `curArr`. opts are applied to both `preArr` and `curArr`.
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

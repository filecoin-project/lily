package diff

import (
	"context"

	"github.com/filecoin-project/go-hamt-ipld/v3"
	adt2 "github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/adt"
)

// Hamt returns a set of changes that transform `preMap` into `curMap`. opts are applied to both `preMap` and `curMap`.
func Hamt(ctx context.Context, preMap, curMap adt2.Map, preStore, curStore adt.Store, hamtOpts ...hamt.Option) ([]*hamt.Change, error) {
	preRoot, err := preMap.Root()
	if err != nil {
		return nil, err
	}

	curRoot, err := curMap.Root()
	if err != nil {
		return nil, err
	}

	return hamt.Diff(ctx, preStore, curStore, preRoot, curRoot, hamtOpts...)
}

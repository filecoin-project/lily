package miner

import (
	"context"

	"github.com/filecoin-project/go-amt-ipld/v4"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

type Changes struct {
	Added    []*typegen.Deferred
	Modified []*ChangeDiff
	Removed  []*typegen.Deferred
}

type ChangeDiff struct {
	Previous *typegen.Deferred
	Current  *typegen.Deferred
}

func DiffPreCommitsDeferred(ctx context.Context, store adt.Store, parent, child State) (*Changes, error) {
	parentMap, err := parent.PrecommitsMap()
	if err != nil {
		return nil, err
	}
	childMap, err := child.PrecommitsMap()
	if err != nil {
		return nil, err
	}
	changes, err := diff.Hamt(ctx, parentMap, childMap, store, store, hamt.UseHashFunction(parent.PrecommitsMapHashFunction()), hamt.UseTreeBitWidth(parent.PrecommitsMapBitWidth()))
	if err != nil {
		return nil, err
	}
	out := &Changes{
		Added:    make([]*typegen.Deferred, 0, len(changes)),
		Modified: make([]*ChangeDiff, 0, len(changes)),
		Removed:  make([]*typegen.Deferred, 0, len(changes)),
	}
	for _, change := range changes {
		switch change.Type {
		case hamt.Add:
			out.Added = append(out.Added, change.After)
		case hamt.Modify:
			out.Modified = append(out.Modified, &ChangeDiff{
				Previous: change.Before,
				Current:  change.After,
			})
		case hamt.Remove:
			out.Removed = append(out.Removed, change.Before)
		}
	}
	return out, nil
}

func DiffSectorsDeferred(ctx context.Context, store adt.Store, parent, child State) (*Changes, error) {
	parentArray, err := parent.SectorsArray()
	if err != nil {
		return nil, err
	}
	childArray, err := child.SectorsArray()
	if err != nil {
		return nil, err
	}
	changes, err := diff.Amt(ctx, parentArray, childArray, store, store, amt.UseTreeBitWidth(uint(parent.SectorsAmtBitwidth())))
	if err != nil {
		return nil, err
	}
	out := &Changes{
		Added:    make([]*typegen.Deferred, 0, len(changes)),
		Modified: make([]*ChangeDiff, 0, len(changes)),
		Removed:  make([]*typegen.Deferred, 0, len(changes)),
	}
	for _, change := range changes {
		switch change.Type {
		case amt.Add:
			out.Added = append(out.Added, change.After)
		case amt.Modify:
			out.Modified = append(out.Modified, &ChangeDiff{
				Previous: change.Before,
				Current:  change.After,
			})
		case amt.Remove:
			out.Removed = append(out.Removed, change.Before)
		}
	}
	return out, nil
}

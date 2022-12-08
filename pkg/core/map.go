package core

import (
	"context"

	"github.com/filecoin-project/go-hamt-ipld/v3"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

type MapDiff struct {
	Added    []*MapChange
	Modified []*MapModification
	Removed  []*MapChange
}

func (m *MapDiff) Size() int {
	return len(m.Added) + len(m.Removed) + len(m.Modified)
}

type MapModification struct {
	Key      string
	Previous typegen.Deferred
	Current  typegen.Deferred
}

type MapChange struct {
	Key   string
	Value typegen.Deferred
}

func DiffMap(ctx context.Context, store adt.Store, child, parent adt.Map, childOpts, parentOpts *adt.MapOpts) (*MapDiff, error) {
	// TODO handle different bitwidth and handFunctions
	if !childOpts.Equal(parentOpts) {
		panic("here")
	}
	changes, err := diff.Hamt(ctx, parent, child, store, store, hamt.UseHashFunction(hamt.HashFunc(childOpts.HashFunc)), hamt.UseTreeBitWidth(childOpts.Bitwidth))
	if err != nil {
		return nil, err
	}
	out := &MapDiff{
		Added:    make([]*MapChange, 0, len(changes)),
		Modified: make([]*MapModification, 0, len(changes)),
		Removed:  make([]*MapChange, 0, len(changes)),
	}
	for _, change := range changes {
		switch change.Type {
		case hamt.Add:
			out.Added = append(out.Added, &MapChange{
				Key:   change.Key,
				Value: *change.After,
			})
		case hamt.Remove:
			out.Removed = append(out.Removed, &MapChange{
				Key:   change.Key,
				Value: *change.Before,
			})
		case hamt.Modify:
			out.Modified = append(out.Modified, &MapModification{
				Key:      change.Key,
				Previous: *change.Before,
				Current:  *change.After,
			})
		}
	}
	return out, nil
}

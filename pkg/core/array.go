package core

import (
	"context"

	"github.com/filecoin-project/go-amt-ipld/v4"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

type ArrayDiff struct {
	Added    []*ArrayChange
	Modified []*ArrayModification
	Removed  []*ArrayChange
}

func (m *ArrayDiff) Size() int {
	return len(m.Added) + len(m.Removed) + len(m.Modified)
}

type ArrayModification struct {
	Key      uint64
	Previous typegen.Deferred
	Current  typegen.Deferred
}

type ArrayChange struct {
	Key   uint64
	Value typegen.Deferred
}

func DiffArray(ctx context.Context, store adt.Store, child, parent adt.Array, childBw, parentBw int) (*ArrayDiff, error) {
	// TODO handle different bitwidth
	if childBw != parentBw {
		panic("here")
	}
	changes, err := diff.Amt(ctx, parent, child, store, store, amt.UseTreeBitWidth(uint(childBw)))
	if err != nil {
		return nil, err
	}
	out := &ArrayDiff{
		Added:    make([]*ArrayChange, 0, len(changes)),
		Modified: make([]*ArrayModification, 0, len(changes)),
		Removed:  make([]*ArrayChange, 0, len(changes)),
	}
	for _, change := range changes {
		switch change.Type {
		case amt.Add:
			out.Added = append(out.Added, &ArrayChange{
				Key:   change.Key,
				Value: *change.After,
			})
		case amt.Remove:
			out.Removed = append(out.Removed, &ArrayChange{
				Key:   change.Key,
				Value: *change.Before,
			})
		case amt.Modify:
			out.Modified = append(out.Modified, &ArrayModification{
				Key:      change.Key,
				Previous: *change.Before,
				Current:  *change.After,
			})
		}
	}
	return out, nil
}

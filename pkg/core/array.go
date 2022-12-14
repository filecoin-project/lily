package core

import (
	"context"

	"github.com/filecoin-project/go-amt-ipld/v4"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

type ArrayModification struct {
	Key      uint64
	Type     ChangeType
	Previous *typegen.Deferred
	Current  *typegen.Deferred
}

type ArrayModifications []*ArrayModification

func DiffArray(ctx context.Context, store adt.Store, child, parent adt.Array, childBw, parentBw int) (ArrayModifications, error) {
	if childBw != parentBw {
		diffContainer := &GenericArrayDiff{
			Added:    []*ArrayModification{},
			Modified: []*ArrayModification{},
			Removed:  []*ArrayModification{},
		}
		if err := diff.CompareArray(child, parent, diffContainer); err != nil {
			return nil, err
		}
		return diffContainer.AsArrayModifications()
	}
	changes, err := diff.Amt(ctx, parent, child, store, store, amt.UseTreeBitWidth(uint(childBw)))
	if err != nil {
		return nil, err
	}
	out := make(ArrayModifications, len(changes))
	for i, change := range changes {
		out[i] = &ArrayModification{
			Key:      change.Key,
			Type:     amtChangeTypeToGeneric(change.Type),
			Previous: change.Before,
			Current:  change.After,
		}
	}
	return out, nil
}

func amtChangeTypeToGeneric(c amt.ChangeType) ChangeType {
	switch c {
	case amt.Add:
		return ChangeTypeAdd
	case amt.Remove:
		return ChangeTypeRemove
	case amt.Modify:
		return ChangeTypeModify
	}
	panic("developer error")
}

type GenericArrayDiff struct {
	Added    ArrayModifications
	Modified ArrayModifications
	Removed  ArrayModifications
}

func (t *GenericArrayDiff) AsArrayModifications() (ArrayModifications, error) {
	out := make(ArrayModifications, len(t.Added)+len(t.Removed)+len(t.Modified))
	idx := 0
	for _, change := range t.Added {
		out[idx] = &ArrayModification{
			Key:      change.Key,
			Type:     ChangeTypeAdd,
			Previous: change.Previous,
			Current:  change.Current,
		}
		idx++
	}
	for _, change := range t.Removed {
		out[idx] = &ArrayModification{
			Key:      change.Key,
			Type:     ChangeTypeRemove,
			Previous: change.Previous,
			Current:  change.Current,
		}
		idx++
	}
	for _, change := range t.Modified {
		out[idx] = &ArrayModification{
			Key:      change.Key,
			Type:     ChangeTypeModify,
			Previous: change.Previous,
			Current:  change.Current,
		}
		idx++
	}
	return out, nil
}

var _ diff.ArrayDiffer = &GenericArrayDiff{}

func (t *GenericArrayDiff) Add(key uint64, val *typegen.Deferred) error {
	t.Added = append(t.Added, &ArrayModification{
		Key:      key,
		Type:     ChangeTypeAdd,
		Previous: nil,
		Current:  val,
	})
	return nil
}

func (t *GenericArrayDiff) Modify(key uint64, from, to *typegen.Deferred) error {
	t.Modified = append(t.Added, &ArrayModification{
		Key:      key,
		Type:     ChangeTypeModify,
		Previous: from,
		Current:  to,
	})
	return nil
}

func (t *GenericArrayDiff) Remove(key uint64, val *typegen.Deferred) error {
	t.Removed = append(t.Added, &ArrayModification{
		Key:      key,
		Type:     ChangeTypeRemove,
		Previous: val,
		Current:  nil,
	})
	return nil
}

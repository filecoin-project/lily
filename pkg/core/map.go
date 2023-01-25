package core

import (
	"context"

	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

type MapModification struct {
	Key      []byte
	Type     ChangeType
	Previous *typegen.Deferred
	Current  *typegen.Deferred
}

type MapModifications []*MapModification

func DiffMap(ctx context.Context, store adt.Store, child, parent adt.Map, childOpts, parentOpts *adt.MapOpts) (MapModifications, error) {
	if !childOpts.Equal(parentOpts) {
		diffContainer := &GenericMapDiff{
			Added:    []*MapModification{},
			Modified: []*MapModification{},
			Removed:  []*MapModification{},
		}
		log.Warn("diffing array using slow comparison")
		if err := diff.CompareMap(child, parent, diffContainer); err != nil {
			return nil, err
		}
		return diffContainer.AsMapModifications()
	}
	changes, err := diff.Hamt(ctx, parent, child, store, store, hamt.UseHashFunction(hamt.HashFunc(childOpts.HashFunc)), hamt.UseTreeBitWidth(childOpts.Bitwidth))
	if err != nil {
		return nil, err
	}
	out := make(MapModifications, len(changes))
	for i, change := range changes {
		out[i] = &MapModification{
			Key:      []byte(change.Key),
			Type:     hamtChangeTypeToGeneric(change.Type),
			Previous: change.Before,
			Current:  change.After,
		}
	}
	return out, nil
}

func hamtChangeTypeToGeneric(c hamt.ChangeType) ChangeType {
	switch c {
	case hamt.Add:
		return ChangeTypeAdd
	case hamt.Remove:
		return ChangeTypeRemove
	case hamt.Modify:
		return ChangeTypeModify
	}
	panic("developer error")
}

type GenericMapDiff struct {
	Added    MapModifications
	Modified MapModifications
	Removed  MapModifications
}

func (t *GenericMapDiff) AsMapModifications() (MapModifications, error) {
	out := make(MapModifications, len(t.Added)+len(t.Removed)+len(t.Modified))
	idx := 0
	for _, change := range t.Added {
		out[idx] = &MapModification{
			Key:      change.Key,
			Type:     ChangeTypeAdd,
			Previous: change.Previous,
			Current:  change.Current,
		}
		idx++
	}
	for _, change := range t.Removed {
		out[idx] = &MapModification{
			Key:      change.Key,
			Type:     ChangeTypeRemove,
			Previous: change.Previous,
			Current:  change.Current,
		}
		idx++
	}
	for _, change := range t.Modified {
		out[idx] = &MapModification{
			Key:      change.Key,
			Type:     ChangeTypeModify,
			Previous: change.Previous,
			Current:  change.Current,
		}
		idx++
	}
	return out, nil
}

var _ diff.MapDiffer = &GenericMapDiff{}

// An adt.Map key that just preserves the underlying string.
type StringKey string

func (k StringKey) Key() string {
	return string(k)
}

func (t *GenericMapDiff) AsKey(key string) (abi.Keyer, error) {
	return StringKey(key), nil
}

func (t *GenericMapDiff) Add(key string, val *typegen.Deferred) error {
	t.Added = append(t.Added, &MapModification{
		Key:      []byte(key),
		Type:     ChangeTypeAdd,
		Previous: nil,
		Current:  val,
	})
	return nil
}

func (t *GenericMapDiff) Modify(key string, from, to *typegen.Deferred) error {
	t.Modified = append(t.Added, &MapModification{
		Key:      []byte(key),
		Type:     ChangeTypeModify,
		Previous: from,
		Current:  to,
	})
	return nil
}

func (t *GenericMapDiff) Remove(key string, val *typegen.Deferred) error {
	t.Removed = append(t.Added, &MapModification{
		Key:      []byte(key),
		Type:     ChangeTypeRemove,
		Previous: val,
		Current:  nil,
	})
	return nil
}

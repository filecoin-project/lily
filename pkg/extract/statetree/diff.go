package statetree

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/lotus/chain/types"
	states0 "github.com/filecoin-project/specs-actors/actors/states"
	states2 "github.com/filecoin-project/specs-actors/v2/actors/states"
	states3 "github.com/filecoin-project/specs-actors/v3/actors/states"
	states4 "github.com/filecoin-project/specs-actors/v4/actors/states"
	states5 "github.com/filecoin-project/specs-actors/v5/actors/states"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
	"github.com/filecoin-project/lily/pkg/core"
)

type ActorDiff struct {
	Executed   *types.Actor
	Current    *types.Actor
	ChangeType core.ChangeType
}

func ActorChanges(ctx context.Context, store adt.Store, current, executed *types.TipSet) (map[address.Address]ActorDiff, error) {
	// we have this special method here to get the HAMT node root required by the faster diffing logic. I hate this.
	oldRoot, oldVersion, err := getStateTreeHamtRootCIDAndVersion(ctx, store, executed.ParentState())
	if err != nil {
		return nil, err
	}
	newRoot, newVersion, err := getStateTreeHamtRootCIDAndVersion(ctx, store, current.ParentState())
	if err != nil {
		return nil, err
	}

	if oldVersion != newVersion {
		panic("here")
	}

	changes, err := diff.Hamt(ctx, oldRoot, newRoot, store, store, hamt.UseTreeBitWidth(5), hamt.UseHashFunction(func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}))
	if err != nil {
		return nil, err
	}
	buf := bytes.NewReader(nil)
	out := map[address.Address]ActorDiff{}
	for _, change := range changes {
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, fmt.Errorf("address in state tree was not valid: %w", err)
		}
		ch := ActorDiff{
			Executed:   new(types.Actor),
			Current:    new(types.Actor),
			ChangeType: core.ChangeTypeUnknown,
		}
		switch change.Type {
		case hamt.Add:
			ch.ChangeType = core.ChangeTypeAdd
			buf.Reset(change.After.Raw)
			err = ch.Current.UnmarshalCBOR(buf)
			buf.Reset(nil)
			if err != nil {
				return nil, err
			}

		case hamt.Remove:
			ch.ChangeType = core.ChangeTypeRemove
			buf.Reset(change.Before.Raw)
			err = ch.Executed.UnmarshalCBOR(buf)
			buf.Reset(nil)
			if err != nil {
				return nil, err
			}

		case hamt.Modify:
			ch.ChangeType = core.ChangeTypeModify
			buf.Reset(change.Before.Raw)
			err = ch.Executed.UnmarshalCBOR(buf)
			buf.Reset(nil)
			if err != nil {
				return nil, err
			}

			buf.Reset(change.After.Raw)
			err = ch.Current.UnmarshalCBOR(buf)
			buf.Reset(nil)
			if err != nil {
				return nil, err
			}
		}
		out[addr] = ch
	}
	return out, nil
}

func getStateTreeHamtRootCIDAndVersion(ctx context.Context, store adt.Store, c cid.Cid) (adt.Map, types.StateTreeVersion, error) {
	var root types.StateRoot
	// Try loading as a new-style state-tree (version/actors tuple).
	if err := store.Get(ctx, c, &root); err != nil {
		// We failed to decode as the new version, must be an old version.
		root.Actors = c
		root.Version = types.StateTreeVersion0
	}

	switch root.Version {
	case types.StateTreeVersion0:
		var tree *states0.Tree
		tree, err := states0.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion1:
		var tree *states2.Tree
		tree, err := states2.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion2:
		var tree *states3.Tree
		tree, err := states3.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion3:
		var tree *states4.Tree
		tree, err := states4.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion4:
		var tree *states5.Tree
		tree, err := states5.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	default:
		return nil, 0, fmt.Errorf("unsupported state tree version: %d", root.Version)
	}
}

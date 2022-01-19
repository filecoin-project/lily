package diff

import (
	"bytes"
	"context"
	"crypto/sha256"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

// ChangeType denotes type of state change
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = iota
	ChangeTypeAdd
	ChangeTypeRemove
	ChangeTypeModify
)

type ActorStateChange struct {
	Actor      types.Actor
	ChangeType ChangeType
}

type ActorStateChangeDiff map[string]ActorStateChange

func GetActorStateChanges(ctx context.Context, store adt.Store, current, previous *types.TipSet) (ActorStateChangeDiff, error) {
	if current.Height() == 0 {
		return GetGenesisActors(ctx, store, current)
	}

	oldTree, err := state.LoadStateTree(store, previous.ParentState())
	if err != nil {
		return nil, err
	}
	oldTreeRoot, err := oldTree.Flush(ctx)
	if err != nil {
		return nil, err
	}

	newTree, err := state.LoadStateTree(store, current.ParentState())
	if err != nil {
		return nil, err
	}
	newTreeRoot, err := oldTree.Flush(ctx)
	if err != nil {
		return nil, err
	}

	if newTree.Version() == oldTree.Version() && (newTree.Version() != types.StateTreeVersion0 && newTree.Version() != types.StateTreeVersion1) {
		changes, err := fastDiff(ctx, store, oldTreeRoot, newTreeRoot)
		if err == nil {
			return changes, nil
		}
		// TODO log error
	}
	actors, err := state.Diff(ctx, oldTree, newTree)
	if err != nil {
		return nil, err
	}

	out := map[string]ActorStateChange{}
	for addr, act := range actors {
		out[addr] = ActorStateChange{
			Actor:      act,
			ChangeType: ChangeTypeUnknown,
		}
	}
	return out, nil
}

func GetGenesisActors(ctx context.Context, store adt.Store, genesis *types.TipSet) (ActorStateChangeDiff, error) {
	out := map[string]ActorStateChange{}
	tree, err := state.LoadStateTree(store, genesis.ParentState())
	if err != nil {
		return nil, err
	}
	if err := tree.ForEach(func(addr address.Address, act *types.Actor) error {
		out[addr.String()] = ActorStateChange{
			Actor:      *act,
			ChangeType: ChangeTypeAdd,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func fastDiff(ctx context.Context, store adt.Store, oldR, newR cid.Cid) (ActorStateChangeDiff, error) {
	// TODO: replace hamt.UseTreeBitWidth and hamt.UseHashFunction with values based on network version
	changes, err := hamt.Diff(ctx, store, store, oldR, newR, hamt.UseTreeBitWidth(5), hamt.UseHashFunction(func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}))
	if err == nil {
		buf := bytes.NewReader(nil)
		out := map[string]ActorStateChange{}
		for _, change := range changes {
			addr, err := address.NewFromBytes([]byte(change.Key))
			if err != nil {
				return nil, xerrors.Errorf("address in state tree was not valid: %w", err)
			}
			var ch ActorStateChange
			switch change.Type {
			case hamt.Add:
				ch.ChangeType = ChangeTypeAdd
				buf.Reset(change.After.Raw)
				err = ch.Actor.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
			case hamt.Remove:
				ch.ChangeType = ChangeTypeRemove
				buf.Reset(change.Before.Raw)
				err = ch.Actor.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
			case hamt.Modify:
				ch.ChangeType = ChangeTypeModify
				buf.Reset(change.After.Raw)
				err = ch.Actor.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
			}
			out[addr.String()] = ch
		}
		return out, nil
	}
	return nil, err
}

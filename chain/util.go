package chain

import (
	"context"
	"time"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	states0 "github.com/filecoin-project/specs-actors/actors/states"
	states2 "github.com/filecoin-project/specs-actors/v2/actors/states"
	states3 "github.com/filecoin-project/specs-actors/v3/actors/states"
	states4 "github.com/filecoin-project/specs-actors/v4/actors/states"
	states5 "github.com/filecoin-project/specs-actors/v5/actors/states"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func getStateTreeMapCIDAndVersion(ctx context.Context, store adt.Store, c cid.Cid) (cid.Cid, types.StateTreeVersion, error) {
	var root types.StateRoot
	// Try loading as a new-style state-tree (version/actors tuple).
	if err := store.Get(ctx, c, &root); err != nil {
		// We failed to decode as the new version, must be an old version.
		root.Actors = c
		root.Version = types.StateTreeVersion0
	}

	var (
		treeMap adt.Map
		err     error
	)
	switch root.Version {
	case types.StateTreeVersion0:
		var tree *states0.Tree
		tree, err = states0.LoadTree(store, root.Actors)
		if tree != nil {
			treeMap = tree.Map
		}
	case types.StateTreeVersion1:
		var tree *states2.Tree
		tree, err = states2.LoadTree(store, root.Actors)
		if tree != nil {
			treeMap = tree.Map
		}
	case types.StateTreeVersion2:
		var tree *states3.Tree
		tree, err = states3.LoadTree(store, root.Actors)
		if tree != nil {
			treeMap = tree.Map
		}
	case types.StateTreeVersion3:
		var tree *states4.Tree
		tree, err = states4.LoadTree(store, root.Actors)
		if tree != nil {
			treeMap = tree.Map
		}
	case types.StateTreeVersion4:
		var tree *states5.Tree
		tree, err = states5.LoadTree(store, root.Actors)
		if tree != nil {
			treeMap = tree.Map
		}
	default:
		return cid.Undef, 0, xerrors.Errorf("unsupported state tree version: %d", root.Version)
	}
	if err != nil {
		log.Errorf("failed to load state tree: %s", err)
		return cid.Undef, 0, xerrors.Errorf("failed to load state tree: %w", err)
	}
	hamtRoot, err := treeMap.Root()
	if err != nil {
		return cid.Undef, 0, err
	}
	return hamtRoot, root.Version, nil
}

const MainnetGenesisTs = 1598306400 // unix timestamp of genesis epoch

// HeightToUnix converts a chain height to a unix timestamp given the unix timestamp of the genesis epoch.
func HeightToUnix(height int64, genesisTs int64) int64 {
	return height*builtin.EpochDurationSeconds + genesisTs
}

// UnixToHeight converts a unix timestamp a chain height given the unix timestamp of the genesis epoch.
func UnixToHeight(ts int64, genesisTs int64) int64 {
	return (ts - genesisTs) / builtin.EpochDurationSeconds
}

// CurrentMainnetHeight calculates the current height of the filecoin mainnet.
func CurrentMainnetHeight() int64 {
	return UnixToHeight(time.Now().Unix(), MainnetGenesisTs)
}

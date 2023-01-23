package extract

import (
	"context"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/tasks"
)

type ChainState struct {
	NetworkName    string
	NetworkVersion uint64
	ActorVersion   uint64
	Message        *MessageStateChanges
	Actors         *ActorStateChanges
}

func State(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*ChainState, error) {
	var (
		blockmessages *MessageStateChanges
		actorChanges  *ActorStateChanges
		err           error
	)

	networkName, err := api.NetworkName(ctx)
	if err != nil {
		return nil, err
	}

	networkVersion, err := api.NetworkVersion(ctx, current.Key())
	if err != nil {
		return nil, err
	}

	actorVersion, err := actortypes.VersionForNetwork(network.Version(networkVersion))
	if err != nil {
		return nil, err
	}

	grp, grpCtx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		// all blocks, messages, implicit messages, from executed and receipts from current
		blockmessages, err = FullBlockMessages(grpCtx, api, current, executed)
		return err
	})
	grp.Go(func() error {
		// all actor changes between current and parent, actor state exists in current.
		actorChanges, err = Actors(grpCtx, api, current, executed, actorVersion)
		return err
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	return &ChainState{
		NetworkName:    networkName,
		NetworkVersion: uint64(networkVersion),
		ActorVersion:   uint64(actorVersion),
		Message:        blockmessages,
		Actors:         actorChanges,
	}, nil
}

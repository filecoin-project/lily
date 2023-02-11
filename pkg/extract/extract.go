package extract

import (
	"context"
	"runtime"
	"time"

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
	Current        *types.TipSet
	Parent         *types.TipSet
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
		start := time.Now()
		// all blocks, messages, implicit messages, from executed and receipts from current
		blockmessages, err = FullBlockMessages(grpCtx, api, current, executed)
		if err != nil {
			log.Errorw("failed to extract full block messages", "error", err)
			return err
		}
		log.Infow("extracted full block messages", "duration", time.Since(start))
		return nil
	})
	grp.Go(func() error {
		start := time.Now()
		// all actor changes between current and parent, actor state exists in current.
		actorChanges, err = Actors(ctx, api, current, executed, runtime.NumCPU())
		if err != nil {
			log.Errorw("failed to extract actor states", "error", err)
			return err
		}
		log.Infow("extracted actor states", "duration", time.Since(start))
		return nil
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	return &ChainState{
		NetworkName:    networkName,
		NetworkVersion: uint64(networkVersion),
		ActorVersion:   uint64(actorVersion),
		Current:        current,
		Parent:         executed,
		Message:        blockmessages,
		Actors:         actorChanges,
	}, nil
}

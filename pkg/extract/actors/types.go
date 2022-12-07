package actors

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/tasks"
)

type ActorDiffer interface {
	Diff(ctx context.Context, api tasks.DataSource, act *ActorChange) (ActorStateChange, error)
}

type ActorStateKind string

type ActorStateChange interface {
	Kind() ActorStateKind
}

type ActorChange struct {
	Address  address.Address
	Executed *types.Actor
	Current  *types.Actor
	Type     core.ChangeType
}

type ActorChanges []*ActorChange

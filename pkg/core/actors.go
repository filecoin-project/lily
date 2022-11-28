package core

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/extract/core")

func ExtractActorChanges(ctx context.Context, api tasks.DataSource, ts, pts *types.TipSet) (ActorChanges, error) {
	actorChanges, err := api.ActorStateChanges(ctx, ts, pts)
	if err != nil {
		return nil, err
	}
	log.Infow("got actors with state change", "count", len(actorChanges), "current", ts.Height(), "parent", pts.Height())

	panic("TODO")
}

// ChangeType denotes type of state change
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = iota
	ChangeTypeAdd
	ChangeTypeRemove
	ChangeTypeModify
)

func (c ChangeType) String() string {
	switch c {
	case ChangeTypeUnknown:
		return "unknown"
	case ChangeTypeAdd:
		return "add"
	case ChangeTypeRemove:
		return "remove"
	case ChangeTypeModify:
		return "modify"
	}
	panic("unreachable")
}

type ActorChange struct {
	Address address.Address
	Actor   *types.Actor
	Type    ChangeType
}

type ActorChanges []*ActorChange

func (ac ActorChanges) WithCID(c cid.Cid) []*ActorChange {
	out := make([]*ActorChange, 0, len(ac))
	for _, change := range ac {
		if change.Actor.Code.Equals(c) {
			out = append(out, change)
		}
	}
	return out
}

func (ac ActorChanges) WithAddress(a address.Address) (*ActorChange, bool) {
	for _, change := range ac {
		if change.Address == a {
			return change, true
		}
	}
	return nil, false
}

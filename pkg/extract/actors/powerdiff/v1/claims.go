package v1

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/power")

type ClaimsChange struct {
	Miner    []byte            `cborgen:"miner"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

type ClaimsChangeList []*ClaimsChange

const KindPowerClaims = "power_claims"

func (c ClaimsChangeList) Kind() actors.ActorStateKind {
	return KindPowerClaims
}

func (p ClaimsChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range p {
		if err := node.Put(core.StringKey(l.Miner), l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

type Claims struct{}

func (Claims) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindPowerClaims, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffClaims(ctx, api, act)
}

func DiffClaims(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, PowerStateLoader, PowerClaimsMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(ClaimsChangeList, len(mapChange))
	for i, change := range mapChange {
		out[i] = &ClaimsChange{
			Miner:    change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		}
	}
	return out, nil
}

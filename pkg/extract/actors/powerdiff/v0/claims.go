package v0

import (
	"context"
	"time"

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
	Miner    []byte
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   core.ChangeType
}

type ClaimsChangeList []*ClaimsChange

const KindPowerClaims = "power_claims"

func (c ClaimsChangeList) Kind() actors.ActorStateKind {
	return KindPowerClaims
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

package v9

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

var log = logging.Logger("lily/extract/actors/balance/v9")

type BalanceChange struct {
	Client   []byte
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   core.ChangeType
}

type BalanceChangeList []*BalanceChange

const KindDataCapBalance = "datacap_balance"

func (b BalanceChangeList) Kind() actors.ActorStateKind {
	return KindDataCapBalance
}

type Balance struct{}

func (Balance) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindDataCapBalance, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffBalances(ctx, api, act)
}

func DiffBalances(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (BalanceChangeList, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, DatacapStateLoader, DatacapBalancesMapLoader)
	if err != nil {
		return nil, err
	}

	out := make(BalanceChangeList, len(mapChange))
	for i, change := range mapChange {
		out[i] = &BalanceChange{
			Client:   change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		}
	}
	return out, nil
}

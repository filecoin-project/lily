package v1

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/extract/actors/balance/v9")

type BalanceChange struct {
	Client   []byte            `cborgen:"client"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

type BalanceChangeList []*BalanceChange

const KindDataCapBalance = "datacap_balance"

func (b BalanceChangeList) Kind() actors.ActorStateKind {
	return KindDataCapBalance
}

func (b BalanceChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range b {
		if err := node.Put(core.StringKey(l.Client), l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

type Balance struct{}

func (b Balance) Type() string {
	return KindDataCapBalance
}

func (Balance) Diff(ctx context.Context, api tasks.DataSource, act *actors.Change) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindDataCapBalance, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffBalances(ctx, api, act)
}

func DiffBalances(ctx context.Context, api tasks.DataSource, act *actors.Change) (BalanceChangeList, error) {
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

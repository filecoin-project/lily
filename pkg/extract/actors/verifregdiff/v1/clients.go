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

// TODO this log name could be confusing since its reused by subsequent versions
var log = logging.Logger("lily/extract/actors/verifreg/v0")

// TODO add cbor gen tags
type ClientsChange struct {
	Client   []byte            `cborgen:"client"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

func (t *ClientsChange) Key() string {
	return core.StringKey(t.Client).Key()
}

type ClientsChangeList []*ClientsChange

const KindVerifregClients = "verifreg_clients"

func (v ClientsChangeList) Kind() actors.ActorStateKind {
	return KindVerifregClients
}

func (v ClientsChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range v {
		if err := node.Put(l, l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

type Clients struct{}

func (Clients) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindVerifregClients, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffClients(ctx, api, act)
}

func DiffClients(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, VerifregStateLoader, VerifiregClientsMapLoader)
	if err != nil {
		return nil, err
	}

	out := make(ClientsChangeList, len(mapChange))
	for i, change := range mapChange {
		out[i] = &ClientsChange{
			Client:   change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		}
	}
	return out, nil

}

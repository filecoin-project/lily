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

// TODO this log name could be confusing since its reused by subsequent versions
var log = logging.Logger("lily/extract/actors/verifreg/v0")

// TODO add cbor gen tags
type ClientsChange struct {
	Client   []byte
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   core.ChangeType
}

type ClientsChangeList []*ClientsChange

const KindVerifregClients = "verifreg_clients"

func (v ClientsChangeList) Kind() actors.ActorStateKind {
	return KindVerifregClients
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

package verifregdiff

import (
	"context"

	"github.com/filecoin-project/go-address"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

type ClientsChange struct {
	Client  address.Address
	DataCap typegen.Deferred
	Change  core.ChangeType
}

type ClientsChangeList []*ClientsChange

const KindVerifregClients = "verifreg_clients"

func (v ClientsChangeList) Kind() actors.ActorStateKind {
	return KindVerifregClients
}

type Clients struct{}

func (Clients) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	return DiffClients(ctx, api, act)
}

func DiffClients(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, VerifregStateLoader, VerifiregClientsMapLoader)
	if err != nil {
		return nil, err
	}

	idx := 0
	out := make(ClientsChangeList, mapChange.Size())
	for _, change := range mapChange.Added {
		// TODO maybe we don't want to marshal these bytes to the address and leave them as bytes in the change struct
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, err
		}
		out[idx] = &ClientsChange{
			Client:  addr,
			DataCap: change.Value,
			Change:  core.ChangeTypeAdd,
		}
		idx++
	}
	for _, change := range mapChange.Removed {
		// TODO maybe we don't want to marshal these bytes to the address and leave them as bytes in the change struct
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, err
		}
		out[idx] = &ClientsChange{
			Client:  addr,
			DataCap: change.Value,
			Change:  core.ChangeTypeRemove,
		}
		idx++
	}
	for _, change := range mapChange.Modified {
		// TODO maybe we don't want to marshal these bytes to the address and leave them as bytes in the change struct
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, err
		}
		out[idx] = &ClientsChange{
			Client:  addr,
			DataCap: change.Current,
			Change:  core.ChangeTypeModify,
		}
		idx++
	}
	return out, nil

}

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

var log = logging.Logger("extract/actors/init")

var _ actors.ActorStateChange = (*AddressChangeList)(nil)

type AddressChange struct {
	Address  []byte
	Current  *typegen.Deferred // actorID
	Previous *typegen.Deferred // actorID
	Change   core.ChangeType
}

type AddressChangeList []*AddressChange

const KindInitAddresses = "init_addresses"

func (a AddressChangeList) Kind() actors.ActorStateKind {
	return KindInitAddresses
}

type Addresses struct{}

func (Addresses) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindInitAddresses, zap.Inline(act), "duration", time.Since(start))
	}()
	return AddressesDiff(ctx, api, act)
}

func AddressesDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, InitStateLoader, InitAddressesMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(AddressChangeList, len(mapChange))
	for i, change := range mapChange {
		out[i] = &AddressChange{
			Address:  change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		}
	}
	return out, nil
}

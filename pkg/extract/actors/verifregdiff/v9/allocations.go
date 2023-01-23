package v9

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	adt2 "github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
	"github.com/filecoin-project/lily/tasks"
)

// TODO add cbor gen tags
type AllocationsChange struct {
	Client   []byte            `cborgen:"provider"`
	ClaimID  []byte            `cborgen:"claimID"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

type AllocationsChangeMap map[address.Address][]*AllocationsChange

const KindVerifregAllocations = "verifreg_allocations"

func (c AllocationsChangeMap) Kind() actors.ActorStateKind {
	return KindVerifregAllocations
}

func (c AllocationsChangeMap) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	topNode, err := adt2.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for client, changes := range c {
		innerNode, err := adt2.MakeEmptyMap(store, bw)
		if err != nil {
			return cid.Undef, err
		}
		for _, change := range changes {
			if err := innerNode.Put(core.StringKey(change.ClaimID), change); err != nil {
				return cid.Undef, err
			}
		}
		innerRoot, err := innerNode.Root()
		if err != nil {
			return cid.Undef, err
		}
		if err := topNode.Put(abi.IdAddrKey(client), typegen.CborCid(innerRoot)); err != nil {
			return cid.Undef, err
		}
	}
	return topNode.Root()
}

type Allocations struct{}

func (Allocations) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindVerifregAllocations, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffAllocations(ctx, api, act)
}

func DiffAllocations(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, v0.VerifregStateLoader, VerifregAllocationMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(AllocationsChangeMap)
	for _, change := range mapChange {
		clientId, err := abi.ParseUIntKey(string(change.Key))
		if err != nil {
			return nil, err
		}
		clientAddress, err := address.NewIDAddress(clientId)
		if err != nil {
			return nil, err
		}
		clientMapChanges, err := diffClientMap(ctx, api, act, clientAddress, change.Key)
		if err != nil {
			return nil, err
		}
		out[clientAddress] = clientMapChanges
	}
	return out, nil
}

func diffClientMap(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, clientAddress address.Address, clientKey []byte) ([]*AllocationsChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, v0.VerifregStateLoader, func(i interface{}) (adt.Map, *adt.MapOpts, error) {
		verifregState := i.(verifreg.State)
		clientAllocationMap, err := verifregState.AllocationMapForClient(clientAddress)
		if err != nil {
			return nil, nil, err
		}
		return clientAllocationMap, &adt.MapOpts{
			Bitwidth: verifregState.AllocationsMapBitWidth(),
			HashFunc: verifregState.AllocationsMapHashFunction(),
		}, nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]*AllocationsChange, 0, len(mapChange))
	for _, change := range mapChange {
		out = append(out, &AllocationsChange{
			Client:   clientKey,
			ClaimID:  change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		})
	}
	return out, nil
}

package v1

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
	"github.com/filecoin-project/lily/chain/actors/builtin/datacap"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

type AllowanceChange struct {
	Owner    []byte            `cborgen:"owner"`
	Operator []byte            `cborgen:"operator"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

type AllowanceChangeMap map[address.Address][]*AllowanceChange

const KindDataCapAllowance = "datacap_allowance"

func (a AllowanceChangeMap) Kind() actors.ActorStateKind {
	return KindDataCapAllowance
}

func (a AllowanceChangeMap) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	topNode, err := adt2.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for owner, changes := range a {
		innerNode, err := adt2.MakeEmptyMap(store, bw)
		if err != nil {
			return cid.Undef, err
		}
		for _, change := range changes {
			if err := innerNode.Put(core.StringKey(change.Operator), change); err != nil {
				return cid.Undef, err
			}
		}
		innerRoot, err := innerNode.Root()
		if err != nil {
			return cid.Undef, err
		}
		if err := topNode.Put(abi.IdAddrKey(owner), typegen.CborCid(innerRoot)); err != nil {
			return cid.Undef, err
		}
	}
	return topNode.Root()
}

type Allowance struct{}

func (Allowance) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindDataCapAllowance, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffAllowances(ctx, api, act)
}

func DiffAllowances(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (AllowanceChangeMap, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, DatacapStateLoader, DatacapAllowancesMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(AllowanceChangeMap)
	for _, change := range mapChange {
		ownerId, err := abi.ParseUIntKey(string(change.Key))
		if err != nil {
			return nil, err
		}
		ownerAddress, err := address.NewIDAddress(ownerId)
		if err != nil {
			return nil, err
		}
		ownerMapChanges, err := diffOwnerMap(ctx, api, act, ownerAddress, change.Key)
		if err != nil {
			return nil, err
		}
		out[ownerAddress] = ownerMapChanges
	}
	return out, nil
}

func diffOwnerMap(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, ownerAddress address.Address, ownerKey []byte) ([]*AllowanceChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, DatacapStateLoader, func(i interface{}) (adt.Map, *adt.MapOpts, error) {
		datacapState := i.(datacap.State)
		clientAllocationMap, err := datacapState.AllowanceMapForOwner(ownerAddress)
		if err != nil {
			return nil, nil, err
		}
		return clientAllocationMap, &adt.MapOpts{
			Bitwidth: datacapState.AllowanceMapBitWidth(),
			HashFunc: datacapState.AllowanceMapHashFunction(),
		}, nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]*AllowanceChange, 0, len(mapChange))
	for _, change := range mapChange {
		out = append(out, &AllowanceChange{
			Owner:    ownerKey,
			Operator: change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		})
	}
	return out, nil
}

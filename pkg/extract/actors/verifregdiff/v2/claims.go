package v2

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v1"

	adt2 "github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

// TODO add cbor gen tags
type ClaimsChange struct {
	Provider []byte            `cborgen:"provider"`
	ClaimID  []byte            `cborgen:"claimID"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

type ClaimsChangeMap map[address.Address][]*ClaimsChange

const KindVerifregClaims = "verifreg_claims"

func (c ClaimsChangeMap) Kind() actors.ActorStateKind {
	return KindVerifregClaims
}

// returns a HAMT[provider]HAMT[claimID]ClaimsChange
func (c ClaimsChangeMap) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	topNode, err := adt2.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for provider, changes := range c {
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
		if err := topNode.Put(abi.IdAddrKey(provider), typegen.CborCid(innerRoot)); err != nil {
			return cid.Undef, err
		}
	}
	return topNode.Root()
}

type Claims struct{}

func (c Claims) Type() string {
	return KindVerifregClaims
}

func (Claims) Diff(ctx context.Context, api tasks.DataSource, act *actors.Change) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindVerifregClaims, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffClaims(ctx, api, act)
}

func DiffClaims(ctx context.Context, api tasks.DataSource, act *actors.Change) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, v0.VerifregStateLoader, VerifregClaimsMapLoader)
	if err != nil {
		return nil, err
	}
	// map change is keyed on provider address with value adt.Map
	out := make(ClaimsChangeMap)
	for _, change := range mapChange {
		providerID, err := abi.ParseUIntKey(string(change.Key))
		if err != nil {
			return nil, err
		}
		providerAddress, err := address.NewIDAddress(providerID)
		if err != nil {
			return nil, err
		}
		subMapChanges, err := diffProviderMap(ctx, api, act, providerAddress, change.Key)
		if err != nil {
			return nil, err
		}
		out[providerAddress] = subMapChanges
	}
	return out, nil
}

func diffProviderMap(ctx context.Context, api tasks.DataSource, act *actors.Change, providerAddress address.Address, providerKey []byte) ([]*ClaimsChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, v0.VerifregStateLoader, func(i interface{}) (adt.Map, *adt.MapOpts, error) {
		verifregState := i.(verifreg.State)
		providerClaimMap, err := verifregState.ClaimMapForProvider(providerAddress)
		if err != nil {
			return nil, nil, err
		}
		return providerClaimMap, &adt.MapOpts{
			Bitwidth: verifregState.ClaimsMapBitWidth(),
			HashFunc: verifregState.ClaimsMapHashFunction(),
		}, nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]*ClaimsChange, 0, len(mapChange))
	for _, change := range mapChange {
		out = append(out, &ClaimsChange{
			Provider: providerKey,
			ClaimID:  change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		})
	}
	return out, nil
}

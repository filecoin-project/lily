package verifreg

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/chain/diff"
	"github.com/filecoin-project/lily/model"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type ClaimsChange struct {
	Provider []byte
	ClaimID  []byte
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   tasks.ChangeType
}

type ClaimsChangeMap map[address.Address][]*ClaimsChange

var VerifregStateLoader = func(store adt.Store, act *types.Actor) (interface{}, error) {
	return verifreg.Load(store, act)
}
var VerifregClaimsMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	verifregState := m.(verifreg.State)
	claimsMap, err := verifregState.ClaimsMap()
	if err != nil {
		return nil, nil, err
	}
	return claimsMap, &adt.MapOpts{
		Bitwidth: verifregState.ClaimsMapBitWidth(),
		HashFunc: verifregState.ClaimsMapHashFunction(),
	}, nil
}

func diffProviderMap(ctx context.Context, node actorstate.ActorStateAPI, act actorstate.ActorInfo, providerAddress address.Address, providerKey []byte) ([]*ClaimsChange, error) {
	mapChange, err := diff.DiffActorMap(ctx, node, act, VerifregStateLoader, func(i interface{}) (adt.Map, *adt.MapOpts, error) {
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

type ClaimExtractor struct{}

func (c ClaimExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	providerChanges, err := diff.DiffActorMap(ctx, node, a, VerifregStateLoader, VerifregClaimsMapLoader)
	if err != nil {
		return nil, err
	}

	// map change is keyed on provider address with value adt.Map
	claimChanges := make(ClaimsChangeMap)
	for _, change := range providerChanges {
		providerID, err := abi.ParseUIntKey(string(change.Key))
		if err != nil {
			return nil, err
		}
		providerAddress, err := address.NewIDAddress(providerID)
		if err != nil {
			return nil, err
		}
		subMapChanges, err := diffProviderMap(ctx, node, a, providerAddress, change.Key)
		if err != nil {
			return nil, err
		}
		claimChanges[providerAddress] = subMapChanges
	}

	out := verifregmodel.VerifiedRegistryClaimList{}
	for provider, change := range claimChanges {
		for _, claim := range change {
			var v verifreg.Claim
			event := verifregmodel.Added
			if claim.Change == tasks.ChangeTypeAdd || claim.Change == tasks.ChangeTypeModify {
				if err := v.UnmarshalCBOR(bytes.NewReader(claim.Current.Raw)); err != nil {
					return nil, err
				}
				if claim.Change == tasks.ChangeTypeModify {
					event = verifregmodel.Modified
				}
			} else {
				if err := v.UnmarshalCBOR(bytes.NewReader(claim.Previous.Raw)); err != nil {
					return nil, err
				}
				event = verifregmodel.Removed
			}
			client, err := address.NewIDAddress(uint64(v.Client))
			if err != nil {
				return nil, err
			}
			claimID, err := abi.ParseUIntKey(string(claim.ClaimID))
			if err != nil {
				return nil, err
			}
			out = append(out, &verifregmodel.VerifiedRegistryClaim{
				Height:    int64(a.Current.Height()),
				StateRoot: a.Current.ParentState().String(),
				Provider:  provider.String(),
				Client:    client.String(),
				ClaimID:   claimID,
				Data:      v.Data.String(),
				Size:      uint64(v.Size),
				TermMin:   int64(v.TermMin),
				TermMax:   int64(v.TermMax),
				TermStart: int64(v.TermStart),
				Sector:    uint64(v.Sector),
				Event:     event,
			})
		}
	}

	return out, nil
}

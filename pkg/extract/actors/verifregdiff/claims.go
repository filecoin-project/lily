package verifregdiff

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

type ClaimsChange struct {
	Provider address.Address
	ClaimID  uint64
	Claim    typegen.Deferred
	Change   core.ChangeType
}

type ClaimsChangeList []*ClaimsChange

const KindVerifregClaims = "verifreg_claims"

func (c ClaimsChangeList) Kind() actors.ActorStateKind {
	return KindVerifregClaims
}

type Claims struct{}

func (Claims) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	return DiffClaims(ctx, api, act)
}

func DiffClaims(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, VerifregStateLoader, VerifiiregClaimsMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(ClaimsChangeList, 0)
	for _, change := range mapChange.Added {
		providerID, err := abi.ParseUIntKey(change.Key)
		if err != nil {
			return nil, err
		}
		providerAddress, err := address.NewIDAddress(providerID)
		if err != nil {
			return nil, err
		}
		added, err := diffSubMap(ctx, api, act, providerAddress)
		if err != nil {
			return nil, err
		}
		if len(added) > 0 {
			out = append(out, added...)
		}
	}
	for _, change := range mapChange.Removed {
		providerID, err := abi.ParseUIntKey(change.Key)
		if err != nil {
			return nil, err
		}
		providerAddress, err := address.NewIDAddress(providerID)
		if err != nil {
			return nil, err
		}
		removed, err := diffSubMap(ctx, api, act, providerAddress)
		if err != nil {
			return nil, err
		}
		if len(removed) > 0 {
			out = append(out, removed...)
		}
	}
	for _, change := range mapChange.Modified {
		providerID, err := abi.ParseUIntKey(change.Key)
		if err != nil {
			return nil, err
		}
		providerAddress, err := address.NewIDAddress(providerID)
		if err != nil {
			return nil, err
		}
		modified, err := diffSubMap(ctx, api, act, providerAddress)
		if err != nil {
			return nil, err
		}
		if len(modified) > 0 {
			out = append(out, modified...)
		}
	}

	return out, nil
}

func diffSubMap(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, providerID address.Address) ([]*ClaimsChange, error) {
	subMapChange, err := generic.DiffActorMap(ctx, api, act, VerifregStateLoader, func(i interface{}) (adt.Map, *adt.MapOpts, error) {
		providerID := providerID
		verifregState := i.(verifreg.State)
		providerClaimMap, err := verifregState.ClaimMapForProvider(providerID)
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
	out := make([]*ClaimsChange, 0, subMapChange.Size())
	for _, subChange := range subMapChange.Added {
		claimID, err := abi.ParseUIntKey(subChange.Key)
		if err != nil {
			return nil, err
		}
		out = append(out, &ClaimsChange{
			Provider: providerID,
			ClaimID:  claimID,
			Claim:    subChange.Value,
			Change:   core.ChangeTypeAdd,
		})
	}
	for _, subChange := range subMapChange.Removed {
		claimID, err := abi.ParseUIntKey(subChange.Key)
		if err != nil {
			return nil, err
		}
		out = append(out, &ClaimsChange{
			Provider: providerID,
			ClaimID:  claimID,
			Claim:    subChange.Value,
			Change:   core.ChangeTypeRemove,
		})
	}
	for _, subChange := range subMapChange.Modified {
		claimID, err := abi.ParseUIntKey(subChange.Key)
		if err != nil {
			return nil, err
		}
		out = append(out, &ClaimsChange{
			Provider: providerID,
			ClaimID:  claimID,
			Claim:    subChange.Current,
			Change:   core.ChangeTypeModify,
		})
	}

	return out, nil
}

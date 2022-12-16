package v9

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

// TODO add cbor gen tags
type ClaimsChange struct {
	Provider []byte
	ClaimID  []byte
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   core.ChangeType
}

type ClaimsChangeList []*ClaimsChange

const KindVerifregClaims = "verifreg_claims"

func (c ClaimsChangeList) Kind() actors.ActorStateKind {
	return KindVerifregClaims
}

type Claims struct{}

func (Claims) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindVerifregClaims, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffClaims(ctx, api, act)
}

func DiffClaims(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, v0.VerifregStateLoader, v0.VerifiiregClaimsMapLoader)
	if err != nil {
		return nil, err
	}
	// map change is keyed on provider address with value adt.Map
	out := make(ClaimsChangeList, 0, len(mapChange))
	for _, change := range mapChange {
		subMapChanges, err := diffSubMap(ctx, api, act, change.Key)
		if err != nil {
			return nil, err
		}
		out = append(out, subMapChanges...)
	}
	return out, nil
}

func diffSubMap(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, providerKey []byte) ([]*ClaimsChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, v0.VerifregStateLoader, func(i interface{}) (adt.Map, *adt.MapOpts, error) {
		providerID, err := abi.ParseUIntKey(string(providerKey))
		if err != nil {
			return nil, nil, err
		}
		providerAddress, err := address.NewIDAddress(providerID)
		if err != nil {
			return nil, nil, err
		}
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

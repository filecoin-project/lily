package verifregdiff

import (
	"context"

	"github.com/filecoin-project/go-address"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type VerifiersChange struct {
	Verifier address.Address
	DataCap  typegen.Deferred
	Change   core.ChangeType
}

type VerifiersChangeList []*VerifiersChange

const KindVerifrefVerifiers = "verifreg_verifiers"

func (v VerifiersChangeList) Kind() actors.ActorStateKind {
	return KindVerifrefVerifiers
}

type Verifiers struct{}

func (Verifiers) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	return DiffVerifiers(ctx, api, act)
}

func DiffVerifiers(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
	}

	currentVerifreg, err := verifreg.Load(api.Store(), act.Current)
	if err != nil {
		return nil, err
	}

	// the actor was added, everything is new in the current state.
	// NB: since this is a singleton actor it was added at genesis and we'll probably never hit this case on mainnet, a test net could in theory.
	if act.Type == core.ChangeTypeAdd {
		var out VerifiersChangeList
		vm, err := currentVerifreg.VerifiersMap()
		if err != nil {
			return nil, err
		}
		var v typegen.Deferred
		if err := vm.ForEach(&v, func(key string) error {
			// TODO maybe we don't want to marshal these bytes to the address and leave them as bytes in the change struct
			addr, err := address.NewFromBytes([]byte(key))
			if err != nil {
				return err
			}
			out = append(out, &VerifiersChange{
				Verifier: addr,
				DataCap:  v,
				Change:   core.ChangeTypeAdd,
			})
			return nil
		}); err != nil {
			return nil, err
		}
	}

	executedVerifreg, err := verifreg.Load(api.Store(), act.Executed)
	if err != nil {
		return nil, err
	}

	verifierChanges, err := verifreg.DiffVerifiersDeferred(ctx, api.Store(), executedVerifreg, currentVerifreg)
	if err != nil {
		return nil, err
	}

	idx := 0
	out := make(VerifiersChangeList, verifierChanges.Size())
	for _, change := range verifierChanges.Added {
		// TODO maybe we don't want to marshal these bytes to the address and leave them as bytes in the change struct
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, err
		}
		out[idx] = &VerifiersChange{
			Verifier: addr,
			DataCap:  change.Value,
			Change:   core.ChangeTypeAdd,
		}
		idx++
	}
	for _, change := range verifierChanges.Removed {
		// TODO maybe we don't want to marshal these bytes to the address and leave them as bytes in the change struct
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, err
		}
		out[idx] = &VerifiersChange{
			Verifier: addr,
			DataCap:  change.Value,
			Change:   core.ChangeTypeRemove,
		}
		idx++
	}
	for _, change := range verifierChanges.Modified {
		// TODO maybe we don't want to marshal these bytes to the address and leave them as bytes in the change struct
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, err
		}
		out[idx] = &VerifiersChange{
			Verifier: addr,
			DataCap:  change.Current,
			Change:   core.ChangeTypeModify,
		}
		idx++
	}
	return out, nil
}

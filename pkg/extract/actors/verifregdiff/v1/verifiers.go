package v1

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

// TODO add cbor gen tags
type VerifiersChange struct {
	Verifier []byte            `cborgen:"verifier"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

func (t *VerifiersChange) Key() string {
	return core.StringKey(t.Verifier).Key()
}

type VerifiersChangeList []*VerifiersChange

const KindVerifregVerifiers = "verifreg_verifiers"

func (v VerifiersChangeList) Kind() actors.ActorStateKind {
	return KindVerifregVerifiers
}

func (v VerifiersChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range v {
		if err := node.Put(l, l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

type Verifiers struct{}

func (v Verifiers) Type() string {
	return KindVerifregVerifiers
}

func (Verifiers) Diff(ctx context.Context, api tasks.DataSource, act *actors.Change) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindVerifregVerifiers, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffVerifiers(ctx, api, act)
}

func DiffVerifiers(ctx context.Context, api tasks.DataSource, act *actors.Change) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, VerifregStateLoader, VerifiregVerifiersMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(VerifiersChangeList, len(mapChange))
	for i, change := range mapChange {
		out[i] = &VerifiersChange{
			Verifier: change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		}
	}
	return out, nil
}

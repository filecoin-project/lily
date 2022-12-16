package v0

import (
	"context"
	"time"

	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

// TODO add cbor gen tags
type VerifiersChange struct {
	Verifier []byte
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   core.ChangeType
}

type VerifiersChangeList []*VerifiersChange

const KindVerifregVerifiers = "verifreg_verifiers"

func (v VerifiersChangeList) Kind() actors.ActorStateKind {
	return KindVerifregVerifiers
}

type Verifiers struct{}

func (Verifiers) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindVerifregVerifiers, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffVerifiers(ctx, api, act)
}

func DiffVerifiers(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
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

package v9

import (
	"context"
	"time"

	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

// TODO add cbor gen tags
type AllocationsChange struct {
	Provider []byte            `cborgen:"provider"`
	ClaimID  []byte            `cborgen:"claimID"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

type AllocationsChangeList []*AllocationsChange

const KindVerifregAllocations = "verifreg_allocations"

func (c AllocationsChangeList) Kind() actors.ActorStateKind {
	return KindVerifregAllocations
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
	// TODO this will look a lot like the DiffClaims method, except diffing allocations
	// - need to add method to the actor that exposes the allocations HAMT and its sub HAMT AllocationMapForProvider()
	panic("NYI")
}

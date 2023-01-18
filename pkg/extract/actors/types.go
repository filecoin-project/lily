package actors

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/tasks"
)

type ActorDiffResult interface {
	Kind() string
	MarshalStateChange(ctx context.Context, s store.Store) (typegen.CBORMarshaler, error)
}

type ActorDiff interface {
	State(ctx context.Context, api tasks.DataSource, act *ActorChange) (ActorDiffResult, error)
}

type ActorStateDiff interface {
	Diff(ctx context.Context, api tasks.DataSource, act *ActorChange) (ActorStateChange, error)
}

type ActorStateKind string

type ActorStateChange interface {
	Kind() ActorStateKind
}

type ActorChange struct {
	Address  address.Address
	Executed *types.Actor
	Current  *types.Actor
	Type     core.ChangeType
}

func (a ActorChange) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("address", a.Address.String()),
		attribute.String("type", builtin.ActorNameByCode(a.Current.Code)),
		attribute.String("change", a.Type.String()),
	}
}

func (a ActorChange) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range a.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

type ActorChanges []*ActorChange

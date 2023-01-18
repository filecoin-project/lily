package actors

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/processor"
)

type ActorStateChangesIPLD struct {
	DataCapActor  *cid.Cid // DataCap
	InitActor     *cid.Cid // Init
	MarketActor   *cid.Cid // Market
	MinerActors   *cid.Cid // HAMT[address]Miner
	PowerActor    *cid.Cid // Power
	RawActors     *cid.Cid // HAMT[address]Raw
	VerifregActor *cid.Cid // Veriferg
}

func (a *ActorStateChangesIPLD) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		// TODO
	}
}

func (a *ActorStateChangesIPLD) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range a.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

func ProcessActorsStates(ctx context.Context, s store.Store, changes *processor.ActorStateChanges) (*ActorStateChangesIPLD, error) {
	out := &ActorStateChangesIPLD{}

	// TODO DataCap

	if changes.InitActor != nil {
		initRoot, err := PutActorDiffResult(ctx, s, changes.InitActor)
		if err != nil {
			return nil, err
		}
		out.InitActor = &initRoot
	}

	if changes.MarketActor != nil {
		marketRoot, err := PutActorDiffResult(ctx, s, changes.MarketActor)
		if err != nil {
			return nil, err
		}
		out.MarketActor = &marketRoot
	}

	if changes.MinerActors != nil {
		minerRoot, err := PutActorDiffResultMap(ctx, s, changes.MinerActors)
		if err != nil {
			return nil, err
		}
		out.MinerActors = &minerRoot
	}

	if changes.PowerActor != nil {
		powerRoot, err := PutActorDiffResult(ctx, s, changes.PowerActor)
		if err != nil {
			return nil, err
		}
		out.PowerActor = &powerRoot
	}

	if changes.RawActors != nil {
		actorsRoot, err := PutActorDiffResultMap(ctx, s, changes.RawActors)
		if err != nil {
			return nil, err
		}
		out.RawActors = &actorsRoot
	}

	if changes.VerifregActor != nil {
		verifregRoot, err := PutActorDiffResult(ctx, s, changes.VerifregActor)
		if err != nil {
			return nil, err
		}

		out.VerifregActor = &verifregRoot
	}

	return out, nil
}

func PutActorDiffResult(ctx context.Context, s store.Store, result actors.ActorDiffResult) (cid.Cid, error) {
	changes, err := result.MarshalStateChange(ctx, s)
	if err != nil {
		return cid.Undef, err
	}
	return s.Put(ctx, changes)
}

func PutActorDiffResultMap(ctx context.Context, s store.Store, results map[address.Address]actors.ActorDiffResult) (cid.Cid, error) {
	actorHamt, err := adt.MakeEmptyMap(s, 5 /*TODO*/)
	if err != nil {
		return cid.Undef, err
	}
	for addr, change := range results {
		msc, err := change.MarshalStateChange(ctx, s)
		if err != nil {
			return cid.Undef, err
		}

		if err := actorHamt.Put(abi.AddrKey(addr), msc); err != nil {
			return cid.Undef, err
		}
	}
	return actorHamt.Root()
}

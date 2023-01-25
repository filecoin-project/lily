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

	"github.com/filecoin-project/lily/pkg/extract"
	"github.com/filecoin-project/lily/pkg/extract/actors"
)

type ActorStateChangesIPLD struct {
	DataCapActor  *cid.Cid `cborgen:"datacap"`  // DataCap
	InitActor     *cid.Cid `cborgen:"init"`     // Init
	MarketActor   *cid.Cid `cborgen:"market"`   // Market
	MinerActors   *cid.Cid `cborgen:"miner"`    // HAMT[address]Miner
	PowerActor    *cid.Cid `cborgen:"power"`    // Power
	RawActors     *cid.Cid `cborgen:"raw"`      // HAMT[address]Raw
	VerifregActor *cid.Cid `cborgen:"verifreg"` // Veriferg
}

func (a *ActorStateChangesIPLD) Attributes() []attribute.KeyValue {
	var out []attribute.KeyValue
	if a.DataCapActor != nil {
		out = append(out, attribute.String("data_cap_root", a.DataCapActor.String()))
	}
	if a.InitActor != nil {
		out = append(out, attribute.String("init_root", a.InitActor.String()))
	}
	if a.MarketActor != nil {
		out = append(out, attribute.String("market_root", a.MarketActor.String()))
	}
	if a.MinerActors != nil {
		out = append(out, attribute.String("miner_root", a.MinerActors.String()))
	}
	if a.PowerActor != nil {
		out = append(out, attribute.String("power_root", a.PowerActor.String()))
	}
	if a.RawActors != nil {
		out = append(out, attribute.String("raw_root", a.RawActors.String()))
	}
	if a.VerifregActor != nil {
		out = append(out, attribute.String("verifreg_root", a.VerifregActor.String()))
	}
	return out
}

func (a *ActorStateChangesIPLD) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range a.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

func ProcessActorsStates(ctx context.Context, s store.Store, changes *extract.ActorStateChanges) (*ActorStateChangesIPLD, error) {
	out := &ActorStateChangesIPLD{}

	if changes.DatacapActor != nil {
		dcapRoot, err := PutActorDiffResult(ctx, s, changes.DatacapActor)
		if err != nil {
			return nil, err
		}
		out.DataCapActor = &dcapRoot
	}

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

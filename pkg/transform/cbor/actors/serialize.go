package actors

import (
	"context"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/pkg/extract/processor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors/init_"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors/market"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors/miner"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors/power"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors/raw"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors/verifreg"
)

type ActorIPLDContainer struct {
	// TODO this needs to be versioned
	CurrentTipSet  *types.TipSet `cborgen:"current"`
	ExecutedTipSet *types.TipSet `cborgen:"executed"`
	MinerActors    cid.Cid       `cborgen:"miners"`
	VerifregActor  *cid.Cid      `cborgen:"verifreg"`
	ActorStates    cid.Cid       `cborgen:"actors"`
	InitActor      *cid.Cid      `cborgen:"init"`
	MarketActor    cid.Cid       `cborgen:"market"`
	PowerActor     cid.Cid       `cborgen:"power"`
}

func (a *ActorIPLDContainer) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("current", a.CurrentTipSet.Key().String()),
		attribute.String("executed", a.ExecutedTipSet.Key().String()),
		attribute.String("miners", a.MinerActors.String()),
		attribute.String("actors", a.ActorStates.String()),
		attribute.String("market", a.MarketActor.String()),
		attribute.String("power", a.PowerActor.String()),
	}
}

func (a *ActorIPLDContainer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range a.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

func ProcessActorsStates(ctx context.Context, bs blockstore.Blockstore, changes *processor.ActorStateChanges) (*ActorIPLDContainer, error) {
	out := &ActorIPLDContainer{
		CurrentTipSet:  changes.Current,
		ExecutedTipSet: changes.Executed,
	}
	if changes.MinerActors != nil {
		minerRoot, err := miner.HandleChanges(ctx, bs, changes.MinerActors)
		if err != nil {
			return nil, err
		}
		out.MinerActors = minerRoot
	}

	if changes.VerifregActor != nil {
		verifregRoot, err := verifreg.HandleChanges(ctx, bs, changes.VerifregActor)
		if err != nil {
			return nil, err
		}
		out.VerifregActor = &verifregRoot
	}

	if changes.ActorStates != nil {
		actorsRoot, err := raw.HandleChanges(ctx, bs, changes.ActorStates)
		if err != nil {
			return nil, err
		}
		out.ActorStates = actorsRoot
	}

	if changes.InitActor != nil {
		initRoot, err := init_.HandleChanges(ctx, bs, changes.InitActor)
		if err != nil {
			return nil, err
		}
		out.InitActor = &initRoot
	}

	if changes.MarketActor != nil {
		marketRoot, err := market.HandleChange(ctx, bs, changes.MarketActor)
		if err != nil {
			return nil, err
		}
		out.MarketActor = marketRoot
	}

	if changes.PowerActor != nil {
		powerRoot, err := power.HandleChange(ctx, bs, changes.PowerActor)
		if err != nil {
			return nil, err
		}
		out.PowerActor = powerRoot
	}
	return out, nil
}

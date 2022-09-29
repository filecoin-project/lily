package sectorinfo

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	minerex "github.com/filecoin-project/lily/tasks/actorstate/miner"
)

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&SectorInfo{}, Extract)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&SectorInfo{}, supportedActors)
}

var _ v2.LilyModel = (*SectorInfo)(nil)

type SectorInfo struct {
	Height                abi.ChainEpoch
	StateRoot             cid.Cid
	Miner                 address.Address
	SectorNumber          abi.SectorNumber
	SealProof             abi.RegisteredSealProof
	SealedCID             cid.Cid
	DealIDs               []abi.DealID
	Activation            abi.ChainEpoch
	Expiration            abi.ChainEpoch
	DealWeight            abi.DealWeight
	VerifiedDealWeight    abi.DealWeight
	InitialPledge         abi.TokenAmount
	ExpectedDayReward     abi.TokenAmount
	ExpectedStoragePledge abi.TokenAmount
	ReplacedSectorAge     abi.ChainEpoch
	ReplacedDayReward     abi.TokenAmount
	SectorKeyCID          cid.Cid
}

func (t *SectorInfo) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: t.Version(),
		Type:    t.Type(),
		Kind:    v2.ModelActorKind,
	}
}

func (t *SectorInfo) Type() v2.ModelType {
	// eww gross
	return v2.ModelType(reflect.TypeOf(SectorInfo{}).Name())
}

func (t *SectorInfo) Version() v2.ModelVersion {
	return 1
}

func (t *SectorInfo) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func Extract(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	//log.Debugw("extract", zap.String("extractor", "SectorInfoExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "SectorInfoExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := minerex.NewMinerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	var sectors []*miner.SectorOnChainInfo
	if !ec.HasPreviousState() {
		// If the miner doesn't have previous state list all of its current sectors.
		sectors, err = ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, fmt.Errorf("loading miner sectors: %w", err)
		}
	} else {
		// If the miner has previous state compute the list of sectors added in its new state.
		sectorChanges, err := api.DiffSectors(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
		for i := range sectorChanges.Added {
			sectors = append(sectors, &sectorChanges.Added[i])
		}
		for i := range sectorChanges.Extended {
			sectors = append(sectors, &sectorChanges.Extended[i].To)
		}
		for i := range sectorChanges.Snapped {
			sectors = append(sectors, &sectorChanges.Snapped[i].To)
		}
	}

	out := make([]v2.LilyModel, len(sectors))
	for i, sector := range sectors {
		sectorKeyCID := cid.Undef
		if sector.SectorKeyCID != nil {
			sectorKeyCID = *sector.SectorKeyCID
		}
		out[i] = &SectorInfo{
			Height:                current.Height(),
			StateRoot:             current.ParentState(),
			Miner:                 a.Address,
			SectorNumber:          sector.SectorNumber,
			SealProof:             sector.SealProof,
			SealedCID:             sector.SealedCID,
			DealIDs:               sector.DealIDs,
			Activation:            sector.Activation,
			Expiration:            sector.Expiration,
			DealWeight:            sector.DealWeight,
			VerifiedDealWeight:    sector.VerifiedDealWeight,
			InitialPledge:         sector.InitialPledge,
			ExpectedDayReward:     sector.ExpectedDayReward,
			ExpectedStoragePledge: sector.ExpectedStoragePledge,
			ReplacedSectorAge:     sector.ReplacedSectorAge,
			ReplacedDayReward:     sector.ReplacedDayReward,
			SectorKeyCID:          sectorKeyCID,
		}
	}
	return out, nil
}

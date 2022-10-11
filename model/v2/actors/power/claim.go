package power

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	power2 "github.com/filecoin-project/lily/tasks/actorstate/power"
)

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&ClaimedPower{}, ExtractClaimedPower)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range power.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&ClaimedPower{}, supportedActors)

}

var _ v2.LilyModel = (*ClaimedPower)(nil)

type ClaimedPower struct {
	Height               abi.ChainEpoch
	StateRoot            cid.Cid
	Miner                address.Address
	Event                ClaimEvent
	RawBytePower         abi.StoragePower
	QualityAdjustedPower abi.StoragePower
}

type ClaimEvent int64

const (
	Added ClaimEvent = iota
	Modified
	Removed
)

func (t ClaimEvent) String() string {
	switch t {
	case Added:
		return "ADDED"
	case Modified:
		return "MODIFIED"
	case Removed:
		return "REMOVED"
	}
	panic(fmt.Sprintf("unhandled type %d developer error", t))
}

func (m *ClaimedPower) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(ClaimedPower{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *ClaimedPower) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *ClaimedPower) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *ClaimedPower) ToStorageBlock() (block.Block, error) {
	data, err := m.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (m *ClaimedPower) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractClaimedPower(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "ClaimedPower"), zap.Inline(a))

	ec, err := power2.NewPowerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, err
	}
	var claimModel []v2.LilyModel
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachClaim(func(miner address.Address, claim power.Claim) error {
			claimModel = append(claimModel, &ClaimedPower{
				Height:               current.Height(),
				StateRoot:            current.ParentState(),
				Miner:                miner,
				Event:                Added,
				RawBytePower:         claim.RawBytePower,
				QualityAdjustedPower: claim.QualityAdjPower,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return claimModel, nil
	}

	// normal case.
	claimChanges, err := power.DiffClaims(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	for _, newClaim := range claimChanges.Added {
		claimModel = append(claimModel, &ClaimedPower{
			Height:               current.Height(),
			StateRoot:            current.ParentState(),
			Miner:                newClaim.Miner,
			Event:                Added,
			RawBytePower:         newClaim.Claim.RawBytePower,
			QualityAdjustedPower: newClaim.Claim.QualityAdjPower,
		})
	}
	for _, modClaim := range claimChanges.Modified {
		claimModel = append(claimModel, &ClaimedPower{
			Height:               current.Height(),
			StateRoot:            current.ParentState(),
			Miner:                modClaim.Miner,
			Event:                Modified,
			RawBytePower:         modClaim.To.RawBytePower,
			QualityAdjustedPower: modClaim.To.QualityAdjPower,
		})
	}
	for _, rmClaim := range claimChanges.Removed {
		claimModel = append(claimModel, &ClaimedPower{
			Height:               current.Height(),
			StateRoot:            current.ParentState(),
			Miner:                rmClaim.Miner,
			Event:                Removed,
			RawBytePower:         rmClaim.Claim.RawBytePower,
			QualityAdjustedPower: rmClaim.Claim.QualityAdjPower,
		})
	}
	return claimModel, nil
}

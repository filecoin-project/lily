package init_

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/zap"

	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("addressstate")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&AddressState{}, Extract)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range market.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&AddressState{}, supportedActors)
}

type AddressState struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	ID        address.Address
	Address   address.Address
}

func (t *AddressState) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *AddressState) ToStorageBlock() (block.Block, error) {
	data, err := t.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (t *AddressState) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func (t *AddressState) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(AddressState{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (t *AddressState) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func Extract(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("extractor", "InitExtractor"), zap.Inline(a))

	// genesis state.
	if a.Current.Height() == 1 {
		initActorState, err := init_.Load(api.Store(), &a.Actor)
		if err != nil {
			return nil, err
		}

		initActorState.GetState()

		var out []v2.LilyModel
		for _, builtinAddress := range []address.Address{
			builtin.SystemActorAddr, builtin.InitActorAddr,
			builtin.RewardActorAddr, builtin.CronActorAddr, builtin.StoragePowerActorAddr, builtin.StorageMarketActorAddr,
			builtin.VerifiedRegistryActorAddr, builtin.BurntFundsActorAddr,
		} {
			out = append(out, &AddressState{
				Height:    0,
				ID:        builtinAddress,
				Address:   builtinAddress,
				StateRoot: a.Executed.ParentState(),
			})
		}
		if err := initActorState.ForEachActor(func(id abi.ActorID, addr address.Address) error {
			idAddr, err := address.NewIDAddress(uint64(id))
			if err != nil {
				return err
			}
			out = append(out, &AddressState{
				Height:    a.Current.Height(),
				ID:        idAddr,
				Address:   addr,
				StateRoot: a.Current.ParentState(),
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}
	prevActor, err := api.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		return nil, fmt.Errorf("loading previous init actor: %w", err)
	}

	prevState, err := init_.Load(api.Store(), prevActor)
	if err != nil {
		return nil, fmt.Errorf("loading previous init actor state: %w", err)
	}

	curState, err := init_.Load(api.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current init actor state: %w", err)
	}

	addressChanges, err := init_.DiffAddressMap(ctx, api.Store(), prevState, curState)
	if err != nil {
		return nil, fmt.Errorf("diffing init actor state: %w", err)
	}

	out := make([]v2.LilyModel, 0, len(addressChanges.Added)+len(addressChanges.Modified))
	for _, newAddr := range addressChanges.Added {
		out = append(out, &AddressState{
			Height:    a.Current.Height(),
			StateRoot: a.Current.ParentState(),
			ID:        newAddr.ID,
			Address:   newAddr.PK,
		})
	}
	for _, modAddr := range addressChanges.Modified {
		out = append(out, &AddressState{
			Height:    a.Current.Height(),
			StateRoot: a.Current.ParentState(),
			ID:        modAddr.To.ID,
			Address:   modAddr.To.PK,
		})
	}

	return out, nil
}

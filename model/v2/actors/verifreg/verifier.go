package verifreg

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

	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	verifreg2 "github.com/filecoin-project/lily/tasks/actorstate/verifreg"
)

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&Verifier{}, ExtractVerifier)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range verifreg.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&Verifier{}, supportedActors)

}

var _ v2.LilyModel = (*Verifier)(nil)

type Verifier struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	Verifier  address.Address
	Event     VerifiedEvent
	DataCap   abi.StoragePower
}

func (m *Verifier) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(Verifier{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *Verifier) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *Verifier) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *Verifier) ToStorageBlock() (block.Block, error) {
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

func (m *Verifier) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractVerifier(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "Verifier"), zap.Inline(a))

	ec, err := verifreg2.NewVerifiedRegistryExtractorContext(ctx, a, api)
	if err != nil {
		return nil, err
	}

	var verifiers []v2.LilyModel
	// if this is the genesis state extract whatever state it has, there is noting to diff against
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachVerifier(func(addr address.Address, dcap abi.StoragePower) error {
			verifiers = append(verifiers, &Verifier{
				Height:    current.Height(),
				StateRoot: current.ParentState(),
				Verifier:  addr,
				Event:     Added,
				DataCap:   dcap,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return verifiers, nil
	}

	changes, err := verifreg.DiffVerifiers(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, fmt.Errorf("diffing verified registry verifiers: %w", err)
	}

	// a new verifier was added
	for _, change := range changes.Added {
		verifiers = append(verifiers, &Verifier{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Verifier:  change.Address,
			Event:     Added,
			DataCap:   change.DataCap,
		})
	}
	// a verifier was removed
	for _, change := range changes.Removed {
		verifiers = append(verifiers, &Verifier{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Verifier:  change.Address,
			Event:     Removed,
			DataCap:   change.DataCap,
		})
	}
	// an existing verifier's DataCap changed
	for _, change := range changes.Modified {
		verifiers = append(verifiers, &Verifier{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Verifier:  change.After.Address,
			Event:     Modified,
			DataCap:   change.After.DataCap,
		})
	}
	return verifiers, nil
}

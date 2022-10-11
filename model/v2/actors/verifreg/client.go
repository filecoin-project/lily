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
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	verifreg2 "github.com/filecoin-project/lily/tasks/actorstate/verifreg"
)

var log = logging.Logger("verifreg")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&VerifiedClient{}, ExtractVerifiedClient)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range verifreg.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&VerifiedClient{}, supportedActors)

}

type VerifiedEvent int64

const (
	Added VerifiedEvent = iota
	Modified
	Removed
)

func (v VerifiedEvent) String() string {
	switch v {
	case Added:
		return "ADDED"
	case Modified:
		return "MODIFIED"
	case Removed:
		return "REMOVED"
	}
	panic(fmt.Sprintf("unhandled type %d developer error", v))
}

var _ v2.LilyModel = (*VerifiedClient)(nil)

type VerifiedClient struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	Event     VerifiedEvent
	Client    address.Address
	DataCap   abi.StoragePower
}

func (m *VerifiedClient) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(VerifiedClient{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *VerifiedClient) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *VerifiedClient) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *VerifiedClient) ToStorageBlock() (block.Block, error) {
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

func (m *VerifiedClient) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractVerifiedClient(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "VerifiedClient"), zap.Inline(a))

	ec, err := verifreg2.NewVerifiedRegistryExtractorContext(ctx, a, api)
	if err != nil {
		return nil, err
	}

	var clients []v2.LilyModel
	// if this is the genesis state extract whatever state it has, there is noting to diff against
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachClient(func(addr address.Address, dcap abi.StoragePower) error {
			clients = append(clients, &VerifiedClient{
				Height:    current.Height(),
				StateRoot: current.ParentState(),
				Client:    addr,
				DataCap:   dcap,
				Event:     Added,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return clients, nil
	}

	changes, err := verifreg.DiffVerifiedClients(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, fmt.Errorf("diffing verified registry clients: %w", err)
	}

	// TODO: we could record the current values of the clients datacap here to allow query of current datacap rather than require
	// the query consider the full chain in aggregation.
	for _, change := range changes.Added {
		clients = append(clients, &VerifiedClient{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Client:    change.Address,
			DataCap:   change.DataCap,
			Event:     Added,
		})
	}
	for _, change := range changes.Modified {
		clients = append(clients, &VerifiedClient{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Client:    change.After.Address,
			DataCap:   change.After.DataCap,
			Event:     Modified,
		})
	}
	for _, change := range changes.Removed {
		clients = append(clients, &VerifiedClient{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Client:    change.Address,
			DataCap:   change.DataCap,
			Event:     Removed,
		})
	}
	return clients, nil
}

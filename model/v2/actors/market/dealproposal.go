package market

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

	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	marketex "github.com/filecoin-project/lily/tasks/actorstate/market"
)

var log = logging.Logger("market")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&DealProposal{}, ExtractDealProposal)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range market.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&DealProposal{}, supportedActors)
}

var _ v2.LilyModel = (*DealProposal)(nil)

type DealProposal struct {
	Height               abi.ChainEpoch
	StateRoot            cid.Cid
	DealID               abi.DealID
	PieceCID             cid.Cid
	PieceSize            abi.PaddedPieceSize
	VerifiedDeal         bool
	Client               address.Address
	Provider             address.Address
	Label                DealLabel
	StartEpoch           abi.ChainEpoch
	EndEpoch             abi.ChainEpoch
	StoragePricePerEpoch abi.TokenAmount
	ProviderCollateral   abi.TokenAmount
	ClientCollateral     abi.TokenAmount
}
type DealLabel struct {
	Label    []byte
	IsString bool
}

func (p *DealProposal) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(DealProposal{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (p *DealProposal) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    p.Height,
		StateRoot: p.StateRoot,
	}
}

func (t *DealProposal) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *DealProposal) ToStorageBlock() (block.Block, error) {
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

func (t *DealProposal) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractDealProposal(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("extractor", "DealProposalExtractor"), zap.Inline(a))

	ec, err := marketex.NewMarketStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, err
	}

	var dealProposals []market.ProposalIDState
	// if this is genesis iterator actors current state.
	if ec.IsGenesis() {
		currDealProposals, err := ec.CurrState.Proposals()
		if err != nil {
			return nil, fmt.Errorf("loading current market deal proposals: %w", err)
		}

		if err := currDealProposals.ForEach(func(id abi.DealID, dp market.DealProposal) error {
			dealProposals = append(dealProposals, market.ProposalIDState{
				ID:       id,
				Proposal: dp,
			})
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		// else diff the actor against previous state and collect any additions that occurred.
		changed, err := ec.CurrState.ProposalsChanged(ec.PrevState)
		if err != nil {
			return nil, fmt.Errorf("checking for deal proposal changes: %w", err)
		}
		if !changed {
			return nil, nil
		}

		changes, err := market.DiffDealProposals(ctx, ec.Store, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, fmt.Errorf("diffing deal proposals: %w", err)
		}

		for _, change := range changes.Added {
			dealProposals = append(dealProposals, market.ProposalIDState{
				ID:       change.ID,
				Proposal: change.Proposal,
			})
		}
	}

	out := make([]v2.LilyModel, len(dealProposals))
	for idx, add := range dealProposals {
		isString := add.Proposal.Label.IsString()
		var label []byte
		if isString {
			tmp, err := add.Proposal.Label.ToString()
			if err != nil {
				return nil, err
			}
			label = []byte(tmp)
		} else {
			tmp, err := add.Proposal.Label.ToBytes()
			if err != nil {
				return nil, err
			}
			label = tmp
		}
		out[idx] = &DealProposal{
			Height:       current.Height(),
			StateRoot:    current.ParentState(),
			DealID:       add.ID,
			PieceCID:     add.Proposal.PieceCID,
			PieceSize:    add.Proposal.PieceSize,
			VerifiedDeal: add.Proposal.VerifiedDeal,
			Client:       add.Proposal.Client,
			Provider:     add.Proposal.Provider,
			Label: DealLabel{
				Label:    label,
				IsString: isString,
			},
			StartEpoch:           add.Proposal.StartEpoch,
			EndEpoch:             add.Proposal.EndEpoch,
			StoragePricePerEpoch: add.Proposal.StoragePricePerEpoch,
			ProviderCollateral:   add.Proposal.ProviderCollateral,
			ClientCollateral:     add.Proposal.ClientCollateral,
		}
	}
	return out, nil
}

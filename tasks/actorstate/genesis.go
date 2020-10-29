package actorstate

import (
	"context"
	"strconv"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	init_ "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	initmodel "github.com/filecoin-project/sentinel-visor/model/actors/init"
	marketmodel "github.com/filecoin-project/sentinel-visor/model/actors/market"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
	powermodel "github.com/filecoin-project/sentinel-visor/model/actors/power"
	genesismodel "github.com/filecoin-project/sentinel-visor/model/genesis"
	"github.com/filecoin-project/sentinel-visor/storage"
)

func NewGenesisProcessor(d *storage.Database, node lens.API) *GenesisProcessor {
	return &GenesisProcessor{
		node:    node,
		storage: d,
	}
}

// GenesisProcessor is a task that processes the genesis block
type GenesisProcessor struct {
	node    lens.API
	storage *storage.Database
}

func (p *GenesisProcessor) ProcessGenesis(ctx context.Context, gen *types.TipSet) error {
	ctx, span := global.Tracer("").Start(ctx, "GenesisProcessor.ProcessGenesis")
	defer span.End()

	genesisAddrs, err := p.node.StateListActors(ctx, gen.Key())
	if err != nil {
		return xerrors.Errorf("list actors: %w", err)
	}

	minerExtractor := StorageMinerExtractor{}
	powerExtractor := StoragePowerExtractor{}

	result := &genesismodel.ProcessGenesisSingletonResult{}
	for _, addr := range genesisAddrs {
		genesisAct, err := p.node.StateGetActor(ctx, addr, gen.Key())
		if err != nil {
			return xerrors.Errorf("get actor: %w", err)
		}

		switch genesisAct.Code {
		case builtin.SystemActorCodeID:
			// TODO
		case builtin.InitActorCodeID:
			res, err := p.initActorState(ctx, gen, genesisAct)
			if err != nil {
				return xerrors.Errorf("init actor state: %w", err)
			}
			result.SetInitActor(res)
		case builtin.CronActorCodeID:
			// TODO
		case builtin.AccountActorCodeID:
			// TODO
		case builtin.StoragePowerActorCodeID:
			res, err := powerExtractor.Extract(ctx, ActorInfo{
				Actor:           *genesisAct,
				Address:         addr,
				ParentStateRoot: gen.ParentState(),
				Epoch:           gen.Height(),
				TipSet:          gen.Key(),
				ParentTipSet:    gen.Parents(),
			}, p.node)
			if err != nil {
				return xerrors.Errorf("power actor state: %w", err)
			}
			// TODO simplify the result to a slice of persistables in follow on.
			result.SetPower(res.(*powermodel.PowerTaskResult))

		case builtin.StorageMarketActorCodeID:
			res, err := p.storageMarketState(ctx, gen)
			if err != nil {
				return xerrors.Errorf("storage market actor state: %w", err)
			}
			result.SetMarket(res)

		case builtin.StorageMinerActorCodeID:
			res, err := minerExtractor.Extract(ctx, ActorInfo{
				Actor:           *genesisAct,
				Address:         addr,
				ParentStateRoot: gen.ParentState(),
				Epoch:           gen.Height(),
				TipSet:          gen.Key(),
				ParentTipSet:    gen.Parents(),
			}, p.node)
			if err != nil {
				return xerrors.Errorf("storage miner actor state: %w", err)
			}
			// TODO simplify the result to a slice of persistables in follow on.
			result.AddMiner(res.(*minermodel.MinerTaskResult))

		case builtin.PaymentChannelActorCodeID:
			// TODO
		case builtin.MultisigActorCodeID:
			// TODO
		case builtin.RewardActorCodeID:
			// TODO
		case builtin.VerifiedRegistryActorCodeID:
			// TODO
		default:
			log.Warnf("unknown actor in genesis state. address: %s code: %s head: %s", addr, genesisAct.Code, genesisAct.Head)
		}
	}

	if err := result.Persist(ctx, p.storage.DB); err != nil {
		return xerrors.Errorf("persist genesis: %w", err)
	}

	return nil
}

func (p *GenesisProcessor) initActorState(ctx context.Context, gen *types.TipSet, act *types.Actor) (*genesismodel.GenesisInitActorTaskResult, error) {
	initActorState, err := init_.Load(p.node.Store(), act)
	if err != nil {
		return nil, err
	}

	out := initmodel.IdAddressList{}
	if err := initActorState.ForEachActor(func(id abi.ActorID, addr address.Address) error {
		out = append(out, &initmodel.IdAddress{
			ID:        id.String(),
			Address:   addr.String(),
			StateRoot: gen.ParentState().String(),
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return &genesismodel.GenesisInitActorTaskResult{AddressMap: out}, nil
}

func (p *GenesisProcessor) storageMarketState(ctx context.Context, gen *types.TipSet) (*genesismodel.GenesisMarketTaskResult, error) {
	dealStates, err := p.node.StateMarketDeals(ctx, gen.Key())
	if err != nil {
		return nil, err
	}

	states := make(marketmodel.MarketDealStates, len(dealStates))
	proposals := make(marketmodel.MarketDealProposals, len(dealStates))
	idx := 0
	for idStr, deal := range dealStates {
		dealID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return nil, err
		}
		states[idx] = &marketmodel.MarketDealState{
			Height:           int64(gen.Height()),
			DealID:           dealID,
			SectorStartEpoch: int64(deal.State.SectorStartEpoch),
			LastUpdateEpoch:  int64(deal.State.LastUpdatedEpoch),
			SlashEpoch:       int64(deal.State.SlashEpoch),
			StateRoot:        gen.ParentState().String(),
		}
		proposals[idx] = &marketmodel.MarketDealProposal{
			Height:               int64(gen.Height()),
			DealID:               dealID,
			StateRoot:            gen.ParentState().String(),
			PaddedPieceSize:      uint64(deal.Proposal.PieceSize),
			UnpaddedPieceSize:    uint64(deal.Proposal.PieceSize.Unpadded()),
			StartEpoch:           int64(deal.Proposal.StartEpoch),
			EndEpoch:             int64(deal.Proposal.EndEpoch),
			ClientID:             deal.Proposal.Client.String(),
			ProviderID:           deal.Proposal.Provider.String(),
			ClientCollateral:     deal.Proposal.ClientCollateral.String(),
			ProviderCollateral:   deal.Proposal.ProviderCollateral.String(),
			StoragePricePerEpoch: deal.Proposal.StoragePricePerEpoch.String(),
			PieceCID:             deal.Proposal.PieceCID.String(),
			IsVerified:           deal.Proposal.VerifiedDeal,
			Label:                deal.Proposal.Label,
		}
		idx++
	}
	return &genesismodel.GenesisMarketTaskResult{
		DealModels:     states,
		ProposalModels: proposals,
	}, nil
}

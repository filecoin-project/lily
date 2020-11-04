package actorstate

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	maddr "github.com/multiformats/go-multiaddr"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
)

// was services/processor/tasks/miner/miner.go

// StorageMinerExtractor extracts miner actor state
type StorageMinerExtractor struct{}

func init() {
	Register(sa0builtin.StorageMinerActorCodeID, StorageMinerExtractor{})
	Register(sa2builtin.StorageMinerActorCodeID, StorageMinerExtractor{})
}

func (m StorageMinerExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	// TODO all processing below can, and probably should, be done in parallel.
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	minerInfoModel, err := ExtractMinerInfo(a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner info: %w", err)
	}

	lockedFundsModel, err := ExtractMinerLockedFunds(a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner locked funds: %w", err)
	}

	feeDebtModel, err := ExtractMinerFeeDebt(a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner fee debt: %w", err)
	}

	currDeadlineModel, err := ExtractMinerCurrentDeadlineInfo(a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner current deadline info: %w", err)
	}

	preCommitModel, sectorModel, sectorDealsModel, sectorEventsModel, err := ExtractMinerSectorData(ctx, ec, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner sector changes: %w", err)
	}

	posts, err := ExtractMinerPoSts(ctx, &a, ec, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner posts: %v", err)
	}

	return &minermodel.MinerTaskResult{
		Posts: posts,

		MinerInfoModel:           minerInfoModel,
		LockedFundsModel:         lockedFundsModel,
		FeeDebtModel:             feeDebtModel,
		CurrentDeadlineInfoModel: currDeadlineModel,
		SectorDealsModel:         sectorDealsModel,
		SectorEventsModel:        sectorEventsModel,
		SectorsModel:             sectorModel,
		PreCommitsModel:          preCommitModel,
	}, nil
}

func NewMinerStateExtractionContext(ctx context.Context, a ActorInfo, node ActorStateAPI) (*MinerStateExtractionContext, error) {
	curActor, err := node.StateGetActor(ctx, a.Address, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner actor: %w", err)
	}

	curTipset, err := node.ChainGetTipSet(ctx, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current tipset: %w", err)
	}

	curState, err := miner.Load(node.Store(), curActor)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner state: %w", err)
	}

	prevState := curState
	if a.Epoch != 0 {
		prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet)
		if err != nil {
			return nil, xerrors.Errorf("loading previous miner %s at tipset %s epoch %d: %w", a.Address, a.ParentTipSet, a.Epoch)
		}

		prevState, err = miner.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous miner actor state: %w", err)
		}
	}

	return &MinerStateExtractionContext{
		PrevState: prevState,
		CurrActor: curActor,
		CurrState: curState,
		CurrTs:    curTipset,
	}, nil
}

type MinerStateExtractionContext struct {
	PrevState miner.State

	CurrActor *types.Actor
	CurrState miner.State
	CurrTs    *types.TipSet
}

func (m *MinerStateExtractionContext) IsGenesis() bool {
	return 0 == m.CurrTs.Height()
}

func ExtractMinerInfo(a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerInfo, error) {
	if ec.IsGenesis() {
		// genesis state special case.
	} else if changed, err := ec.CurrState.MinerInfoChanged(ec.PrevState); err != nil {
		return nil, err
	} else if !changed {
		return nil, nil
	}
	// miner info has changed.

	newInfo, err := ec.CurrState.Info()
	if err != nil {
		return nil, err
	}

	var newWorker string
	if newInfo.NewWorker != address.Undef {
		newWorker = newInfo.NewWorker.String()
	}

	var newCtrlAddresses []string
	for _, addr := range newInfo.ControlAddresses {
		newCtrlAddresses = append(newCtrlAddresses, addr.String())
	}

	// best effort to decode, we have no control over what miners put in this field, its just bytes.
	var newMultiAddrs []string
	for _, addr := range newInfo.Multiaddrs {
		newMaddr, err := maddr.NewMultiaddrBytes(addr)
		if err == nil {
			newMultiAddrs = append(newMultiAddrs, newMaddr.String())
		} else {
			log.Debugw("failed to decode miner multiaddr", "miner", a.Address, "multiaddress", addr, "error", err)
		}
	}
	return &minermodel.MinerInfo{
		Height:                  int64(ec.CurrTs.Height()),
		MinerID:                 a.Address.String(),
		StateRoot:               a.ParentStateRoot.String(),
		OwnerID:                 newInfo.Owner.String(),
		WorkerID:                newInfo.Worker.String(),
		NewWorker:               newWorker,
		WorkerChangeEpoch:       int64(newInfo.WorkerChangeEpoch),
		ConsensusFaultedElapsed: int64(newInfo.ConsensusFaultElapsed),
		PeerID:                  newInfo.PeerId.String(),
		ControlAddresses:        newCtrlAddresses,
		MultiAddresses:          newMultiAddrs,
	}, nil
}

func ExtractMinerLockedFunds(a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerLockedFund, error) {
	currLocked, err := ec.CurrState.LockedFunds()
	if err != nil {
		return nil, xerrors.Errorf("loading current miner locked funds: %w", err)
	}
	if !ec.IsGenesis() {
		prevLocked, err := ec.PrevState.LockedFunds()
		if err != nil {
			return nil, xerrors.Errorf("loading previous miner locked funds: %w", err)
		}
		if prevLocked == currLocked {
			return nil, nil
		}
	}
	// funds changed

	return &minermodel.MinerLockedFund{
		Height:            int64(ec.CurrTs.Height()),
		MinerID:           a.Address.String(),
		StateRoot:         a.ParentStateRoot.String(),
		LockedFunds:       currLocked.VestingFunds.String(),
		InitialPledge:     currLocked.InitialPledgeRequirement.String(),
		PreCommitDeposits: currLocked.PreCommitDeposits.String(),
	}, nil
}

func ExtractMinerFeeDebt(a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerFeeDebt, error) {
	currDebt, err := ec.CurrState.FeeDebt()
	if err != nil {
		return nil, xerrors.Errorf("loading current miner fee debt: %w", err)
	}

	if !ec.IsGenesis() {
		prevDebt, err := ec.PrevState.FeeDebt()
		if err != nil {
			return nil, xerrors.Errorf("loading previous miner fee debt: %w", err)
		}
		if prevDebt == currDebt {
			return nil, nil
		}
	}
	// debt changed

	return &minermodel.MinerFeeDebt{
		Height:    int64(ec.CurrTs.Height()),
		MinerID:   a.Address.String(),
		StateRoot: a.ParentStateRoot.String(),
		FeeDebt:   currDebt.String(),
	}, nil
}

func ExtractMinerCurrentDeadlineInfo(a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerCurrentDeadlineInfo, error) {
	currDeadlineInfo, err := ec.CurrState.DeadlineInfo(ec.CurrTs.Height())
	if err != nil {
		return nil, err
	}

	if !ec.IsGenesis() {
		prevDeadlineInfo, err := ec.PrevState.DeadlineInfo(ec.CurrTs.Height())
		if err != nil {
			return nil, err
		}
		if prevDeadlineInfo == currDeadlineInfo {
			return nil, nil
		}
	}

	return &minermodel.MinerCurrentDeadlineInfo{
		Height:        int64(ec.CurrTs.Height()),
		MinerID:       a.Address.String(),
		StateRoot:     a.ParentStateRoot.String(),
		DeadlineIndex: currDeadlineInfo.Index,
		PeriodStart:   int64(currDeadlineInfo.PeriodStart),
		Open:          int64(currDeadlineInfo.Open),
		Close:         int64(currDeadlineInfo.Close),
		Challenge:     int64(currDeadlineInfo.Challenge),
		FaultCutoff:   int64(currDeadlineInfo.FaultCutoff),
	}, nil
}

func ExtractMinerSectorData(ctx context.Context, ec *MinerStateExtractionContext, a ActorInfo, node ActorStateAPI) (minermodel.MinerPreCommitInfoList, minermodel.MinerSectorInfoList, minermodel.MinerSectorDealList, minermodel.MinerSectorEventList, error) {
	preCommitChanges := new(miner.PreCommitChanges)
	preCommitChanges.Added = []miner.SectorPreCommitOnChainInfo{}
	preCommitChanges.Removed = []miner.SectorPreCommitOnChainInfo{}

	sectorChanges := new(miner.SectorChanges)
	sectorChanges.Added = []miner.SectorOnChainInfo{}
	sectorChanges.Removed = []miner.SectorOnChainInfo{}
	sectorChanges.Extended = []miner.SectorExtensions{}

	sectorDealsModel := minermodel.MinerSectorDealList{}
	if ec.IsGenesis() {
		msectors, err := ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		sectorChanges.Added = make([]miner.SectorOnChainInfo, len(msectors))
		for idx, sector := range msectors {
			sectorChanges.Added[idx] = *sector
			for _, dealID := range sector.DealIDs {
				sectorDealsModel = append(sectorDealsModel, &minermodel.MinerSectorDeal{
					Height:   int64(ec.CurrTs.Height()),
					MinerID:  a.Address.String(),
					SectorID: uint64(sector.SectorNumber),
					DealID:   uint64(dealID),
				})
			}
		}
	} else { // not genesis state, need to diff with previous state to compute changes.
		var err error
		preCommitChanges, err = miner.DiffPreCommits(ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, nil, nil, nil, xerrors.Errorf("diffing miner precommits: %w", err)
		}

		sectorChanges, err = miner.DiffSectors(ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, nil, nil, nil, xerrors.Errorf("diffing miner sectors: %w", err)
		}

		for _, newSector := range sectorChanges.Added {
			for _, dealID := range newSector.DealIDs {
				sectorDealsModel = append(sectorDealsModel, &minermodel.MinerSectorDeal{
					Height:   int64(ec.CurrTs.Height()),
					MinerID:  a.Address.String(),
					SectorID: uint64(newSector.SectorNumber),
					DealID:   uint64(dealID),
				})
			}

		}
	}
	sectorEventModel, err := extractMinerSectorEvents(ctx, node, a, ec, sectorChanges, preCommitChanges)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// transform the preCommitChanges to a model
	preCommitModel := minermodel.MinerPreCommitInfoList{}
	for _, added := range preCommitChanges.Added {
		pcm := &minermodel.MinerPreCommitInfo{
			Height:    int64(ec.CurrTs.Height()),
			MinerID:   a.Address.String(),
			SectorID:  uint64(added.Info.SectorNumber),
			StateRoot: a.ParentStateRoot.String(),

			SealedCID:       added.Info.SealedCID.String(),
			SealRandEpoch:   int64(added.Info.SealRandEpoch),
			ExpirationEpoch: int64(added.Info.Expiration),

			PreCommitDeposit:   added.PreCommitDeposit.String(),
			PreCommitEpoch:     int64(added.PreCommitEpoch),
			DealWeight:         added.DealWeight.String(),
			VerifiedDealWeight: added.VerifiedDealWeight.String(),

			IsReplaceCapacity:      added.Info.ReplaceCapacity,
			ReplaceSectorDeadline:  added.Info.ReplaceSectorDeadline,
			ReplaceSectorPartition: added.Info.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(added.Info.ReplaceSectorNumber),
		}
		preCommitModel = append(preCommitModel, pcm)
	}

	// transform sector changes to a model
	sectorModel := minermodel.MinerSectorInfoList{}
	for _, added := range sectorChanges.Added {
		sm := &minermodel.MinerSectorInfo{
			Height:                int64(ec.CurrTs.Height()),
			MinerID:               a.Address.String(),
			SectorID:              uint64(added.SectorNumber),
			StateRoot:             a.ParentStateRoot.String(),
			SealedCID:             added.SealedCID.String(),
			ActivationEpoch:       int64(added.Activation),
			ExpirationEpoch:       int64(added.Expiration),
			DealWeight:            added.DealWeight.String(),
			VerifiedDealWeight:    added.VerifiedDealWeight.String(),
			InitialPledge:         added.InitialPledge.String(),
			ExpectedDayReward:     added.ExpectedDayReward.String(),
			ExpectedStoragePledge: added.ExpectedStoragePledge.String(),
		}
		sectorModel = append(sectorModel, sm)
	}
	return preCommitModel, sectorModel, sectorDealsModel, sectorEventModel, nil
}

func ExtractMinerPoSts(ctx context.Context, actor *ActorInfo, ec *MinerStateExtractionContext, node ActorStateAPI) (map[uint64]cid.Cid, error) {
	// short circuit genesis state, no PoSt messages in genesis blocks.
	if ec.IsGenesis() {
		return nil, nil
	}
	posts := make(map[uint64]cid.Cid)
	block := actor.TipSet.Cids()[0]
	msgs, err := node.ChainGetBlockMessages(ctx, block)
	if err != nil {
		return nil, xerrors.Errorf("diffing miner posts: %v", err)
	}

	var partitions map[uint64]miner.Partition
	loadPartitions := func(state miner.State, epoch abi.ChainEpoch) (map[uint64]miner.Partition, error) {
		info, err := state.DeadlineInfo(epoch)
		if err != nil {
			return nil, err
		}
		dline, err := state.LoadDeadline(info.Index)
		if err != nil {
			return nil, err
		}
		pmap := make(map[uint64]miner.Partition)
		if err := dline.ForEachPartition(func(idx uint64, p miner.Partition) error {
			pmap[idx] = p
			return nil
		}); err != nil {
			return nil, err
		}
		return pmap, nil
	}

	processPostMsg := func(msg *types.Message) error {
		sectors := make([]uint64, 0)
		rcpt, err := node.StateGetReceipt(ctx, msg.Cid(), actor.TipSet)
		if err != nil {
			return err
		}
		if rcpt == nil || rcpt.ExitCode.IsError() {
			return nil
		}
		params := miner.SubmitWindowedPoStParams{}
		if err := params.UnmarshalCBOR(bytes.NewBuffer(msg.Params)); err != nil {
			return err
		}

		if partitions == nil {
			partitions, err = loadPartitions(ec.CurrState, ec.CurrTs.Height())
			if err != nil {
				return err
			}
		}

		for _, p := range params.Partitions {
			all, err := partitions[p.Index].AllSectors()
			if err != nil {
				return err
			}
			proven, err := bitfield.SubtractBitField(all, p.Skipped)
			if err != nil {
				return err
			}

			if err := proven.ForEach(func(sector uint64) error {
				sectors = append(sectors, sector)
				return nil
			}); err != nil {
				return err
			}
		}

		for _, s := range sectors {
			posts[s] = msg.Cid()
		}
		return nil
	}

	for _, msg := range msgs.BlsMessages {
		if msg.To == actor.Address && msg.Method == 5 /* miner.SubmitWindowedPoSt */ {
			if err := processPostMsg(msg); err != nil {
				return nil, err
			}
		}
	}
	for _, msg := range msgs.SecpkMessages {
		if msg.Message.To == actor.Address && msg.Message.Method == 5 /* miner.SubmitWindowedPoSt */ {
			if err := processPostMsg(&msg.Message); err != nil {
				return nil, err
			}
		}
	}
	return posts, nil
}

func extractMinerSectorEvents(ctx context.Context, node ActorStateAPI, a ActorInfo, ec *MinerStateExtractionContext, sc *miner.SectorChanges, pc *miner.PreCommitChanges) (minermodel.MinerSectorEventList, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtractor.extractMinerSectorEvents")
	defer span.End()

	ps, err := extractMinerPartitionsDiff(ctx, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner partition diff: %w", err)
	}

	out := minermodel.MinerSectorEventList{}
	sectorAdds := make(map[abi.SectorNumber]miner.SectorOnChainInfo)

	// if there were changes made to the miners partition lists
	if ps != nil {
		// build an index of removed sector expiration's for comparison below.
		removedSectors, err := node.StateMinerSectors(ctx, a.Address, &ps.Removed, a.TipSet)
		if err != nil {
			return nil, xerrors.Errorf("fetching miners removed sectors: %w", err)
		}
		rmExpireIndex := make(map[uint64]abi.ChainEpoch)
		for _, rm := range removedSectors {
			rmExpireIndex[uint64(rm.SectorNumber)] = rm.Expiration
		}

		// track terminated and expired sectors
		if err := ps.Removed.ForEach(func(u uint64) error {
			event := minermodel.SectorTerminated
			expiration := rmExpireIndex[u]
			if expiration == a.Epoch {
				event = minermodel.SectorExpired
			}
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     event,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners removed sectors: %w", err)
		}

		// track recovering sectors
		if err := ps.Recovering.ForEach(func(u uint64) error {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     minermodel.SectorRecovering,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners recovering sectors: %w", err)
		}

		// track faulted sectors
		if err := ps.Faulted.ForEach(func(u uint64) error {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     minermodel.SectorFaulted,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners faulted sectors: %w", err)
		}

		// track recovered sectors
		if err := ps.Recovered.ForEach(func(u uint64) error {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     minermodel.SectorRecovered,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners recovered sectors: %w", err)
		}
	}

	// if there were changes made to the miners sectors list
	if sc != nil {
		// track sector add and commit-capacity add
		for _, add := range sc.Added {
			event := minermodel.SectorAdded
			if len(add.DealIDs) == 0 {
				event = minermodel.CommitCapacityAdded
			}
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  uint64(add.SectorNumber),
				Event:     event,
			})
			sectorAdds[add.SectorNumber] = add
		}

		// track sector extensions
		for _, mod := range sc.Extended {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  uint64(mod.To.SectorNumber),
				Event:     minermodel.SectorExtended,
			})
		}

	}

	// if there were changes made to the miners precommit list
	if pc != nil {
		// track precommit addition
		for _, add := range pc.Added {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  uint64(add.Info.SectorNumber),
				Event:     minermodel.PreCommitAdded,
			})
		}
	}

	return out, nil
}

// PartitionStatus contains bitfileds of sectorID's that are removed, faulted, recovered and recovering.
type PartitionStatus struct {
	Removed    bitfield.BitField
	Faulted    bitfield.BitField
	Recovering bitfield.BitField
	Recovered  bitfield.BitField
}

func extractMinerPartitionsDiff(ctx context.Context, ec *MinerStateExtractionContext) (*PartitionStatus, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtractor.minerPartitionDiff")
	defer span.End()

	// short circuit genesis state.
	if ec.IsGenesis() {
		return nil, nil
	}

	dlDiff, err := miner.DiffDeadlines(ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	if dlDiff == nil {
		return nil, nil
	}

	removed := bitfield.New()
	faulted := bitfield.New()
	recovered := bitfield.New()
	recovering := bitfield.New()

	for _, deadline := range dlDiff {
		for _, partition := range deadline {
			removed, err = bitfield.MergeBitFields(removed, partition.Removed)
			if err != nil {
				return nil, err
			}
			faulted, err = bitfield.MergeBitFields(faulted, partition.Faulted)
			if err != nil {
				return nil, err
			}
			recovered, err = bitfield.MergeBitFields(recovered, partition.Recovered)
			if err != nil {
				return nil, err
			}
			recovering, err = bitfield.MergeBitFields(recovering, partition.Recovering)
			if err != nil {
				return nil, err
			}
		}
	}
	return &PartitionStatus{
		Removed:    removed,
		Faulted:    faulted,
		Recovering: recovering,
		Recovered:  recovered,
	}, nil
}

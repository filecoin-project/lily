package actorstate

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	maddr "github.com/multiformats/go-multiaddr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	miner "github.com/filecoin-project/lily/chain/actors/builtin/miner"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
)

// was services/processor/tasks/miner/miner.go

// StorageMinerExtractor extracts miner actor state
type StorageMinerExtractor struct{}

func init() {
	for _, c := range miner.AllCodes() {
		Register(c, StorageMinerExtractor{})
	}
}

func (m StorageMinerExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "StorageMinerExtractor")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("actor", a.Address.String()))
	}
	defer span.End()

	stop := metrics.Timer(ctx, metrics.StateExtractionDuration)
	defer stop()

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("creating miner state extraction context: %w", err)
	}

	minerInfoModel, err := ExtractMinerInfo(ctx, a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner info: %w", err)
	}

	lockedFundsModel, err := ExtractMinerLockedFunds(ctx, a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner locked funds: %w", err)
	}

	feeDebtModel, err := ExtractMinerFeeDebt(ctx, a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner fee debt: %w", err)
	}

	currDeadlineModel, err := ExtractMinerCurrentDeadlineInfo(ctx, a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner current deadline info: %w", err)
	}

	preCommitModel, sectorModelV7, sectorDealsModel, sectorEventsModel, err := ExtractMinerSectorData(ctx, ec, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner sector changes: %w", err)
	}

	posts, err := ExtractMinerPoSts(ctx, &a, ec, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner posts: %v", err)
	}

	// a wrapper type used to avoid calling persist on nil models
	out := &minermodel.MinerTaskResult{
		Posts:                    posts,
		MinerInfoModel:           minerInfoModel,
		FeeDebtModel:             feeDebtModel,
		LockedFundsModel:         lockedFundsModel,
		CurrentDeadlineInfoModel: currDeadlineModel,
		PreCommitsModel:          preCommitModel,
		SectorEventsModel:        sectorEventsModel,
		SectorDealsModel:         sectorDealsModel,
	}

	// if the miner actor is v1-6 persist its model to the miner_sector_infos table
	var sectorModelV6Minus minermodel.MinerSectorInfoV1_6List
	if a.Actor.Code.Equals(miner0.Actor{}.Code()) ||
		a.Actor.Code.Equals(miner2.Actor{}.Code()) ||
		a.Actor.Code.Equals(miner3.Actor{}.Code()) ||
		a.Actor.Code.Equals(miner4.Actor{}.Code()) ||
		a.Actor.Code.Equals(miner5.Actor{}.Code()) ||
		a.Actor.Code.Equals(miner6.Actor{}.Code()) {
		for _, sm := range sectorModelV7 {
			sectorModelV6Minus = append(sectorModelV6Minus, &minermodel.MinerSectorInfoV1_6{
				Height:                sm.Height,
				MinerID:               sm.MinerID,
				SectorID:              sm.SectorID,
				StateRoot:             sm.StateRoot,
				SealedCID:             sm.SealedCID,
				ActivationEpoch:       sm.ActivationEpoch,
				ExpirationEpoch:       sm.ExpirationEpoch,
				DealWeight:            sm.DealWeight,
				VerifiedDealWeight:    sm.VerifiedDealWeight,
				InitialPledge:         sm.InitialPledge,
				ExpectedDayReward:     sm.ExpectedDayReward,
				ExpectedStoragePledge: sm.ExpectedStoragePledge,
			})
		}
		out.SectorsModelV1_6 = sectorModelV6Minus
	} else {
		// if the miner actor is v7 or newer persist its model the miner_sector_infos_v7 table
		out.SectorsModelV7 = sectorModelV7
	}

	return out, nil
}

func NewMinerStateExtractionContext(ctx context.Context, a ActorInfo, node ActorStateAPI) (*MinerStateExtractionContext, error) {
	ctx, span := otel.Tracer("").Start(ctx, "NewMinerExtractionContext")
	defer span.End()

	curState, err := miner.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner state: %w", err)
	}

	prevTipset := a.TipSet
	prevState := curState
	if a.Epoch != 1 {
		prevTipset = a.ParentTipSet

		prevActor, err := node.Actor(ctx, a.Address, a.ParentTipSet.Key())
		if err != nil {
			// if the actor exists in the current state and not in the parent state then the
			// actor was created in the current state.
			if err == types.ErrActorNotFound {
				return &MinerStateExtractionContext{
					PrevState: prevState,
					PrevTs:    prevTipset,
					CurrActor: &a.Actor,
					CurrState: curState,
					CurrTs:    a.TipSet,
				}, nil
			}
			return nil, xerrors.Errorf("loading previous miner %s at tipset %s epoch %d: %w", a.Address, a.ParentTipSet.Key(), a.Epoch, err)
		}

		prevState, err = miner.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous miner actor state: %w", err)
		}
	}

	return &MinerStateExtractionContext{
		PrevState: prevState,
		PrevTs:    prevTipset,
		CurrActor: &a.Actor,
		CurrState: curState,
		CurrTs:    a.TipSet,
	}, nil
}

type MinerStateExtractionContext struct {
	PrevState miner.State
	PrevTs    *types.TipSet

	CurrActor *types.Actor
	CurrState miner.State
	CurrTs    *types.TipSet
}

func (m *MinerStateExtractionContext) HasPreviousState() bool {
	return !(m.CurrTs.Height() == 1 || m.PrevState == m.CurrState)
}

func ExtractMinerInfo(ctx context.Context, a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerInfo, error) {
	_, span := otel.Tracer("").Start(ctx, "ExtractMinerInfo")
	defer span.End()
	if !ec.HasPreviousState() {
		// means this miner was created in this tipset or genesis special case
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
	mi := &minermodel.MinerInfo{
		Height:                  int64(ec.CurrTs.Height()),
		MinerID:                 a.Address.String(),
		StateRoot:               a.ParentStateRoot.String(),
		OwnerID:                 newInfo.Owner.String(),
		WorkerID:                newInfo.Worker.String(),
		NewWorker:               newWorker,
		WorkerChangeEpoch:       int64(newInfo.WorkerChangeEpoch),
		ConsensusFaultedElapsed: int64(newInfo.ConsensusFaultElapsed),
		ControlAddresses:        newCtrlAddresses,
		MultiAddresses:          newMultiAddrs,
		SectorSize:              uint64(newInfo.SectorSize),
	}

	if newInfo.PeerId != nil {
		mi.PeerID = newInfo.PeerId.String()
	}

	return mi, nil
}

func ExtractMinerLockedFunds(ctx context.Context, a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerLockedFund, error) {
	_, span := otel.Tracer("").Start(ctx, "ExtractMinerLockedFunds")
	defer span.End()
	currLocked, err := ec.CurrState.LockedFunds()
	if err != nil {
		return nil, xerrors.Errorf("loading current miner locked funds: %w", err)
	}
	if ec.HasPreviousState() {
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

func ExtractMinerFeeDebt(ctx context.Context, a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerFeeDebt, error) {
	_, span := otel.Tracer("").Start(ctx, "ExtractMinerFeeDebt")
	defer span.End()
	currDebt, err := ec.CurrState.FeeDebt()
	if err != nil {
		return nil, xerrors.Errorf("loading current miner fee debt: %w", err)
	}

	if ec.HasPreviousState() {
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

func ExtractMinerCurrentDeadlineInfo(ctx context.Context, a ActorInfo, ec *MinerStateExtractionContext) (*minermodel.MinerCurrentDeadlineInfo, error) {
	_, span := otel.Tracer("").Start(ctx, "ExtractMinerDeadlineInfo")
	defer span.End()
	currDeadlineInfo, err := ec.CurrState.DeadlineInfo(ec.CurrTs.Height())
	if err != nil {
		return nil, err
	}

	if ec.HasPreviousState() {
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

func ExtractMinerSectorData(ctx context.Context, ec *MinerStateExtractionContext, a ActorInfo, node ActorStateAPI) (minermodel.MinerPreCommitInfoList, minermodel.MinerSectorInfoV7List, minermodel.MinerSectorDealList, minermodel.MinerSectorEventList, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ExtractMinerSectorData")
	defer span.End()
	preCommitChanges := new(miner.PreCommitChanges)
	preCommitChanges.Added = []miner.SectorPreCommitOnChainInfo{}
	preCommitChanges.Removed = []miner.SectorPreCommitOnChainInfo{}

	sectorChanges := new(miner.SectorChanges)
	sectorChanges.Added = []miner.SectorOnChainInfo{}
	sectorChanges.Removed = []miner.SectorOnChainInfo{}
	sectorChanges.Extended = []miner.SectorModification{}
	sectorChanges.Snapped = []miner.SectorModification{}

	sectorDealsModel := minermodel.MinerSectorDealList{}
	if !ec.HasPreviousState() {
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
		preCommitChanges, err = miner.DiffPreCommits(ctx, node.Store(), ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, nil, nil, nil, xerrors.Errorf("diffing miner precommits: %w", err)
		}

		sectorChanges, err = miner.DiffSectors(ctx, node.Store(), ec.PrevState, ec.CurrState)
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
	sectorModel := minermodel.MinerSectorInfoV7List{}
	for _, added := range sectorChanges.Added {
		sectorKeyCID := ""
		if added.SectorKeyCID != nil {
			sectorKeyCID = added.SectorKeyCID.String()
		}
		sm := &minermodel.MinerSectorInfoV7{
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
			// added in specs-actors v7
			SectorKeyCID: sectorKeyCID,
		}
		sectorModel = append(sectorModel, sm)
	}

	// do the same for extended sectors, since they have a new deadline
	for _, extended := range sectorChanges.Extended {
		sectorKeyCID := ""
		if extended.To.SectorKeyCID != nil {
			sectorKeyCID = extended.To.SectorKeyCID.String()
		}
		sm := &minermodel.MinerSectorInfoV7{
			Height:                int64(ec.CurrTs.Height()),
			MinerID:               a.Address.String(),
			SectorID:              uint64(extended.To.SectorNumber),
			StateRoot:             a.ParentStateRoot.String(),
			SealedCID:             extended.To.SealedCID.String(),
			ActivationEpoch:       int64(extended.To.Activation),
			ExpirationEpoch:       int64(extended.To.Expiration),
			DealWeight:            extended.To.DealWeight.String(),
			VerifiedDealWeight:    extended.To.VerifiedDealWeight.String(),
			InitialPledge:         extended.To.InitialPledge.String(),
			ExpectedDayReward:     extended.To.ExpectedDayReward.String(),
			ExpectedStoragePledge: extended.To.ExpectedStoragePledge.String(),
			// added in specs-actors v7
			SectorKeyCID: sectorKeyCID,
		}
		sectorModel = append(sectorModel, sm)
	}

	// same for snapped sectors, since many fields will have changed:
	// https://github.com/filecoin-project/FIPs/blob/master/FIPS/fip-0019.md#provereplicaupdates-actor-method
	for _, extended := range sectorChanges.Snapped {
		sectorKeyCID := ""
		if extended.To.SectorKeyCID != nil {
			sectorKeyCID = extended.To.SectorKeyCID.String()
		}
		sm := &minermodel.MinerSectorInfoV7{
			Height:                int64(ec.CurrTs.Height()),
			MinerID:               a.Address.String(),
			SectorID:              uint64(extended.To.SectorNumber),
			StateRoot:             a.ParentStateRoot.String(),
			SealedCID:             extended.To.SealedCID.String(),
			ActivationEpoch:       int64(extended.To.Activation),
			ExpirationEpoch:       int64(extended.To.Expiration),
			DealWeight:            extended.To.DealWeight.String(),
			VerifiedDealWeight:    extended.To.VerifiedDealWeight.String(),
			InitialPledge:         extended.To.InitialPledge.String(),
			ExpectedDayReward:     extended.To.ExpectedDayReward.String(),
			ExpectedStoragePledge: extended.To.ExpectedStoragePledge.String(),
			// added in specs-actors v7
			SectorKeyCID: sectorKeyCID,
		}
		sectorModel = append(sectorModel, sm)
	}

	return preCommitModel, sectorModel, sectorDealsModel, sectorEventModel, nil
}

func ExtractMinerPoSts(ctx context.Context, actor *ActorInfo, ec *MinerStateExtractionContext, node ActorStateAPI) (minermodel.MinerSectorPostList, error) {
	_, span := otel.Tracer("").Start(ctx, "ExtractMinerPoSts")
	defer span.End()
	// short circuit genesis state, no PoSt messages in genesis blocks.
	if !ec.HasPreviousState() {
		return nil, nil
	}
	addr := actor.Address.String()
	posts := make(minermodel.MinerSectorPostList, 0)

	var partitions map[uint64]miner.Partition
	loadPartitions := func(state miner.State, epoch abi.ChainEpoch) (map[uint64]miner.Partition, error) {
		info, err := state.DeadlineInfo(epoch)
		if err != nil {
			return nil, xerrors.Errorf("deadline info: %w", err)
		}
		dline, err := state.LoadDeadline(info.Index)
		if err != nil {
			return nil, xerrors.Errorf("load deadline: %w", err)
		}
		pmap := make(map[uint64]miner.Partition)
		if err := dline.ForEachPartition(func(idx uint64, p miner.Partition) error {
			pmap[idx] = p
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("foreach partition: %w", err)
		}
		return pmap, nil
	}

	processPostMsg := func(msg *lens.ExecutedMessage) error {
		sectors := make([]uint64, 0)
		if msg.Receipt == nil || msg.Receipt.ExitCode.IsError() {
			return nil
		}
		params := miner.SubmitWindowedPoStParams{}
		if err := params.UnmarshalCBOR(bytes.NewBuffer(msg.Message.Params)); err != nil {
			return xerrors.Errorf("unmarshal post params: %w", err)
		}

		var err error
		// use previous miner state and tipset state since we are using parent messages
		if partitions == nil {
			partitions, err = loadPartitions(ec.PrevState, ec.PrevTs.Height())
			if err != nil {
				return xerrors.Errorf("load partitions: %w", err)
			}
		}

		for _, p := range params.Partitions {
			all, err := partitions[p.Index].AllSectors()
			if err != nil {
				return xerrors.Errorf("all sectors: %w", err)
			}
			proven, err := bitfield.SubtractBitField(all, p.Skipped)
			if err != nil {
				return xerrors.Errorf("subtract skipped bitfield: %w", err)
			}

			if err := proven.ForEach(func(sector uint64) error {
				sectors = append(sectors, sector)
				return nil
			}); err != nil {
				return xerrors.Errorf("foreach proven: %w", err)
			}
		}

		for _, s := range sectors {
			posts = append(posts, &minermodel.MinerSectorPost{
				Height:         int64(ec.PrevTs.Height()),
				MinerID:        addr,
				SectorID:       s,
				PostMessageCID: msg.Cid.String(),
			})
		}
		return nil
	}

	tsMsgs, err := node.ExecutedAndBlockMessages(ctx, actor.TipSet, actor.ParentTipSet)
	if err != nil {
		return nil, xerrors.Errorf("getting executed and block messages: %w", err)
	}

	for _, msg := range tsMsgs.Executed {
		if msg.Message.To == actor.Address && msg.Message.Method == 5 /* miner.SubmitWindowedPoSt */ {
			if err := processPostMsg(msg); err != nil {
				return nil, xerrors.Errorf("process post msg: %w", err)
			}
		}
	}
	return posts, nil
}

func extractMinerSectorEvents(ctx context.Context, node ActorStateAPI, a ActorInfo, ec *MinerStateExtractionContext, sc *miner.SectorChanges, pc *miner.PreCommitChanges) (minermodel.MinerSectorEventList, error) {
	ctx, span := otel.Tracer("").Start(ctx, "extractMinerSectorEvents")
	defer span.End()

	ps, err := extractMinerPartitionsDiff(ctx, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner partition diff: %w", err)
	}

	out := minermodel.MinerSectorEventList{}

	// if there were changes made to the miners partition lists
	if ps != nil {
		// build an index of removed sector expiration's for comparison below.

		removedSectors, err := ec.CurrState.LoadSectors(&ps.Removed)
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

		// track sectors snapped.
		for _, mod := range sc.Snapped {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  uint64(mod.To.SectorNumber),
				Event:     minermodel.SectorSnapped,
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
	_, span := otel.Tracer("").Start(ctx, "extractMinerPartitionDiff") // nolint: ineffassign,staticcheck
	defer span.End()

	// short circuit genesis state.
	if !ec.HasPreviousState() {
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

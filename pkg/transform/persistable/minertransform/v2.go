package minertransform

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
)

func V2MinerHandler(ctx context.Context, stateDiff *minerdiff.StateDiff) (model.PersistableList, error) {
	var out model.PersistableList
	if stateDiff.InfoChange != nil {
		infoModel, err := V2MinerInfoHandler(ctx, stateDiff)
		if err != nil {
			return nil, err
		}
		out = append(out, infoModel)
	}
	if stateDiff.SectorChanges != nil {
		sectorModel, err := V2MinerSectorHandler(ctx, stateDiff)
		if err != nil {
			return nil, err
		}
		out = append(out, sectorModel)
	}
	if stateDiff.PreCommitChanges != nil {
		preCommitModel, err := V2MinerPreCommitHandler(ctx, stateDiff)
		if err != nil {
			return nil, err
		}
		out = append(out, preCommitModel)
	}
	return out, nil
}

func V2MinerPreCommitHandler(ctx context.Context, stateDiff *minerdiff.StateDiff) (model.Persistable, error) {
	minerPreCommits := make([]*miner2.SectorPreCommitOnChainInfo, len(stateDiff.PreCommitChanges))
	for i, preCommit := range stateDiff.PreCommitChanges {
		var minerPreCommit *miner2.SectorPreCommitOnChainInfo
		if err := minerPreCommit.UnmarshalCBOR(bytes.NewReader(preCommit.PreCommit.Raw)); err != nil {
			return nil, err
		}
		minerPreCommits[i] = minerPreCommit
	}
	preCommitModel := make(minermodel.MinerPreCommitInfoList, len(minerPreCommits))
	for i, preCommit := range minerPreCommits {
		preCommitModel[i] = &minermodel.MinerPreCommitInfo{
			Height:                 int64(stateDiff.TipSet.Height()),
			MinerID:                stateDiff.Miner.Address.String(),
			StateRoot:              stateDiff.TipSet.ParentState().String(),
			SectorID:               uint64(preCommit.Info.SectorNumber),
			SealedCID:              preCommit.Info.SealedCID.String(),
			SealRandEpoch:          int64(preCommit.Info.SealRandEpoch),
			ExpirationEpoch:        int64(preCommit.Info.Expiration),
			PreCommitDeposit:       preCommit.PreCommitDeposit.String(),
			PreCommitEpoch:         int64(preCommit.PreCommitEpoch),
			DealWeight:             preCommit.DealWeight.String(),
			VerifiedDealWeight:     preCommit.VerifiedDealWeight.String(),
			IsReplaceCapacity:      preCommit.Info.ReplaceCapacity,
			ReplaceSectorDeadline:  preCommit.Info.ReplaceSectorDeadline,
			ReplaceSectorPartition: preCommit.Info.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(preCommit.Info.ReplaceSectorNumber),
		}
	}

	return preCommitModel, nil

}

func V2MinerSectorHandler(ctx context.Context, stateDiff *minerdiff.StateDiff) (model.Persistable, error) {
	minerSectors := make([]*miner2.SectorOnChainInfo, len(stateDiff.SectorChanges))
	for i, sector := range stateDiff.SectorChanges {
		var minerSector *miner2.SectorOnChainInfo
		if err := minerSector.UnmarshalCBOR(bytes.NewReader(sector.Sector.Raw)); err != nil {
			return nil, err
		}
		minerSectors[i] = minerSector
	}
	sectorModel := make(minermodel.MinerSectorInfoV1_6List, len(minerSectors))
	for i, sector := range minerSectors {
		sectorModel[i] = &minermodel.MinerSectorInfoV1_6{
			Height:                int64(stateDiff.TipSet.Height()),
			MinerID:               stateDiff.Miner.Address.String(),
			StateRoot:             stateDiff.TipSet.ParentState().String(),
			SectorID:              uint64(sector.SectorNumber),
			SealedCID:             sector.SealedCID.String(),
			ActivationEpoch:       int64(sector.Activation),
			ExpirationEpoch:       int64(sector.Expiration),
			DealWeight:            sector.DealWeight.String(),
			VerifiedDealWeight:    sector.VerifiedDealWeight.String(),
			InitialPledge:         sector.InitialPledge.String(),
			ExpectedDayReward:     sector.ExpectedDayReward.String(),
			ExpectedStoragePledge: sector.ExpectedStoragePledge.String(),
		}
	}
	return sectorModel, nil
}

func V2MinerInfoHandler(ctx context.Context, stateDiff *minerdiff.StateDiff) (model.Persistable, error) {
	var minerInfo miner2.MinerInfo
	if err := minerInfo.UnmarshalCBOR(bytes.NewReader(stateDiff.InfoChange.Info.Raw)); err != nil {
		return nil, err
	}
	var newWorker string
	var newWorkerEpoch int64
	if pendingWorkerKey := minerInfo.PendingWorkerKey; pendingWorkerKey != nil {
		if pendingWorkerKey.NewWorker != address.Undef {
			newWorker = pendingWorkerKey.NewWorker.String()
		}
		newWorkerEpoch = int64(pendingWorkerKey.EffectiveAt)
	}

	var newCtrlAddresses []string
	for _, addr := range minerInfo.ControlAddresses {
		newCtrlAddresses = append(newCtrlAddresses, addr.String())
	}

	// best effort to decode, we have no control over what miners put in this field, its just bytes.
	var newMultiAddrs []string
	for _, addr := range minerInfo.Multiaddrs {
		newMaddr, err := maddr.NewMultiaddrBytes(addr)
		if err == nil {
			newMultiAddrs = append(newMultiAddrs, newMaddr.String())
		} else {
			log.Debugw("failed to decode miner multiaddr", "miner", stateDiff.Miner.Address, "multiaddress", addr, "error", err)
		}
	}
	mi := &minermodel.MinerInfo{
		Height:                  int64(stateDiff.TipSet.Height()),
		MinerID:                 stateDiff.Miner.Address.String(),
		StateRoot:               stateDiff.TipSet.ParentState().String(),
		OwnerID:                 minerInfo.Owner.String(),
		WorkerID:                minerInfo.Worker.String(),
		NewWorker:               newWorker,
		WorkerChangeEpoch:       newWorkerEpoch,
		ConsensusFaultedElapsed: int64(minerInfo.ConsensusFaultElapsed),
		ControlAddresses:        newCtrlAddresses,
		MultiAddresses:          newMultiAddrs,
		SectorSize:              uint64(minerInfo.SectorSize),
	}

	if minerInfo.PeerId != nil {
		newPeerID, err := peer.IDFromBytes(minerInfo.PeerId)
		if err != nil {
			log.Warnw("failed to decode miner peerID", "miner", stateDiff.Miner.Address, "head", stateDiff.Miner.Actor.Head.String(), "error", err)
		} else {
			mi.PeerID = newPeerID.String()
		}
	}
	return mi, nil
}

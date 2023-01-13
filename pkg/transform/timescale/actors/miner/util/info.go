package util

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p/core/peer"
	maddr "github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
)

type WorkerKeyChanges interface {
	NewWorker() address.Address
	EffectiveAt() abi.ChainEpoch
}

type MinerInfo interface {
	PendingWorkerKey() (WorkerKeyChanges, bool)
	ControlAddresses() []address.Address
	Multiaddrs() []abi.Multiaddrs
	Owner() address.Address
	Worker() address.Address
	SectorSize() abi.SectorSize
	PeerId() abi.PeerID
}

func ExtractMinerInfo(ctx context.Context, current, executed *types.TipSet, addr address.Address, info MinerInfo) (model.Persistable, error) {
	var newWorker string
	var newWorkerEpoch int64
	if pendingWorkerKey, changed := info.PendingWorkerKey(); changed {
		if pendingWorkerKey.NewWorker() != address.Undef {
			newWorker = pendingWorkerKey.NewWorker().String()
		}
		newWorkerEpoch = int64(pendingWorkerKey.EffectiveAt())
	}

	var newCtrlAddresses []string
	for _, addr := range info.ControlAddresses() {
		newCtrlAddresses = append(newCtrlAddresses, addr.String())
	}

	// best effort to decode, we have no control over what miners put in this field, its just bytes.
	var newMultiAddrs []string
	for _, addr := range info.Multiaddrs() {
		newMaddr, err := maddr.NewMultiaddrBytes(addr)
		if err == nil {
			newMultiAddrs = append(newMultiAddrs, newMaddr.String())
		} else {
			//log.Debugw("failed to decode miner multiaddr", "miner", a.Address, "multiaddress", addr, "error", err)
		}
	}
	mi := &minermodel.MinerInfo{
		Height:                  int64(current.Height()),
		MinerID:                 addr.String(),
		StateRoot:               current.ParentState().String(),
		OwnerID:                 info.Owner().String(),
		WorkerID:                info.Worker().String(),
		NewWorker:               newWorker,
		WorkerChangeEpoch:       newWorkerEpoch,
		ConsensusFaultedElapsed: -1,
		ControlAddresses:        newCtrlAddresses,
		MultiAddresses:          newMultiAddrs,
		SectorSize:              uint64(info.SectorSize()),
	}

	if info.PeerId() != nil {
		newPeerID, err := peer.IDFromBytes(info.PeerId())
		if err != nil {
			//log.Warnw("failed to decode miner peerID", "miner", a.Address, "head", a.Actor.Head.String(), "error", err)
		} else {
			mi.PeerID = newPeerID.String()
		}
	}

	return mi, nil
}

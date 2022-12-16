package v0

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/libp2p/go-libp2p/core/peer"
	maddr "github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
)

func HandleMinerInfo(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *v0.StateDiffResult) (model.Persistable, error) {
	info := change.InfoChange
	var out model.Persistable
	var err error
	if err := core.StateReadDeferred(ctx, info.Info, func(in *miner0.MinerInfo) error {
		out, err = MinerInfoAsModel(ctx, current, executed, addr, *in)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func MinerInfoAsModel(ctx context.Context, current, executed *types.TipSet, addr address.Address, info miner0.MinerInfo) (model.Persistable, error) {
	return GenericMinerInfoAsModel(ctx, current, executed, addr, info)
}

func GenericMinerInfoAsModel(ctx context.Context, current, executed *types.TipSet, addr address.Address, info miner0.MinerInfo) (model.Persistable, error) {
	var newWorker string
	var newWorkerEpoch int64
	if pendingWorkerKey := info.PendingWorkerKey; pendingWorkerKey != nil {
		if pendingWorkerKey.NewWorker != address.Undef {
			newWorker = pendingWorkerKey.NewWorker.String()
		}
		newWorkerEpoch = int64(pendingWorkerKey.EffectiveAt)
	}

	var newCtrlAddresses []string
	for _, addr := range info.ControlAddresses {
		newCtrlAddresses = append(newCtrlAddresses, addr.String())
	}

	// best effort to decode, we have no control over what miners put in this field, its just bytes.
	var newMultiAddrs []string
	for _, addr := range info.Multiaddrs {
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
		OwnerID:                 info.Owner.String(),
		WorkerID:                info.Worker.String(),
		NewWorker:               newWorker,
		WorkerChangeEpoch:       newWorkerEpoch,
		ConsensusFaultedElapsed: -1,
		ControlAddresses:        newCtrlAddresses,
		MultiAddresses:          newMultiAddrs,
		SectorSize:              uint64(info.SectorSize),
	}

	if info.PeerId != nil {
		newPeerID, err := peer.IDFromBytes(info.PeerId)
		if err != nil {
			//log.Warnw("failed to decode miner peerID", "miner", a.Address, "head", a.Actor.Head.String(), "error", err)
		} else {
			mi.PeerID = newPeerID.String()
		}
	}

	return mi, nil
}

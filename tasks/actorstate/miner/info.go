package miner

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("lily/tasks/miner")

type InfoExtractor struct{}

func (InfoExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "InfoExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "InfoExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}
	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	if ec.HasPreviousState() {
		if changed, err := ec.CurrState.MinerInfoChanged(ec.PrevState); err != nil {
			return nil, err
		} else if !changed {
			return nil, nil
		}
	}
	// miner info has changed.

	newInfo, err := ec.CurrState.Info()
	if err != nil {
		return nil, err
	}

	var newWorker string
	var newWorkerEpoch int64
	if pendingWorkerKey := newInfo.PendingWorkerKey; pendingWorkerKey != nil {
		if pendingWorkerKey.NewWorker != address.Undef {
			newWorker = pendingWorkerKey.NewWorker.String()
		}
		newWorkerEpoch = int64(pendingWorkerKey.EffectiveAt)
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
		StateRoot:               a.Current.ParentState().String(),
		OwnerID:                 newInfo.Owner.String(),
		WorkerID:                newInfo.Worker.String(),
		NewWorker:               newWorker,
		WorkerChangeEpoch:       newWorkerEpoch,
		ConsensusFaultedElapsed: int64(newInfo.ConsensusFaultElapsed),
		ControlAddresses:        newCtrlAddresses,
		MultiAddresses:          newMultiAddrs,
		SectorSize:              uint64(newInfo.SectorSize),
	}

	if newInfo.PeerId != nil {
		newPeerID, err := peer.IDFromBytes(newInfo.PeerId)
		if err != nil {
			log.Warnw("failed to decode miner peerID", "miner", a.Address, "head", a.Actor.Head.String(), "error", err)
		} else {
			mi.PeerID = newPeerID.String()
		}
	}

	return mi, nil
}

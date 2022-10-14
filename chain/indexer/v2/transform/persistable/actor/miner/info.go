package miner

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
)

type MinerInfoTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewMinerInfoTransform(taskName string) *MinerInfoTransform {
	i := miner.MinerInfo{}
	return &MinerInfoTransform{meta: i.Meta(), taskName: taskName}
}

func (s *MinerInfoTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerInfoList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				mi := modeldata.(*miner.MinerInfo)
				var newWorker string
				var newWorkerEpoch int64
				if pendingWorkerKey := mi.PendingWorkerKey; pendingWorkerKey != nil {
					if pendingWorkerKey.NewWorker != address.Undef {
						newWorker = pendingWorkerKey.NewWorker.String()
					}
					newWorkerEpoch = int64(pendingWorkerKey.EffectiveAt)
				}
				var newCtrlAddresses []string
				for _, addr := range mi.ControlAddresses {
					newCtrlAddresses = append(newCtrlAddresses, addr.String())
				}
				// best effort to decode, we have no control over what miners put in this field, its just bytes.
				var newMultiAddrs []string
				for _, addr := range mi.Multiaddrs {
					newMaddr, err := maddr.NewMultiaddrBytes(addr)
					if err == nil {
						newMultiAddrs = append(newMultiAddrs, newMaddr.String())
					} else {
						log.Debugw("failed to decode miner multiaddr", "miner", mi.Multiaddrs, "multiaddress", addr, "stateroot", mi.StateRoot, "error", err)
					}
				}
				var newPeerID string
				if mi.PeerID != nil {
					maybePeerID, err := peer.IDFromBytes(mi.PeerID)
					if err != nil {
						log.Warnw("failed to decode miner peerID", "miner", mi.Miner, "stateroot", mi.StateRoot, "error", err)
					} else {
						newPeerID = maybePeerID.String()
					}
				}
				sqlModels = append(sqlModels, &minermodel.MinerInfo{
					Height:                  int64(mi.Height),
					MinerID:                 mi.Miner.String(),
					StateRoot:               mi.StateRoot.String(),
					OwnerID:                 mi.Owner.String(),
					WorkerID:                mi.Worker.String(),
					NewWorker:               newWorker,
					WorkerChangeEpoch:       newWorkerEpoch,
					ConsensusFaultedElapsed: int64(mi.ConsensusFaultElapsed),
					PeerID:                  newPeerID,
					ControlAddresses:        newCtrlAddresses,
					MultiAddresses:          newMultiAddrs,
					SectorSize:              uint64(mi.SectorSize),
				})
			}
			if len(sqlModels) > 0 {
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
		}
	}
	return nil
}

func (s *MinerInfoTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *MinerInfoTransform) Name() string {
	i := MinerInfoTransform{}
	return reflect.TypeOf(i).Name()
}

func (s *MinerInfoTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

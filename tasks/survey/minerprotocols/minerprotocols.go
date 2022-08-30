package minerprotocols

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-address"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/lily/model"
	observed "github.com/filecoin-project/lily/model/surveyed"
)

var log = logging.Logger("lily/task/minerprotocols")

type API interface {
	Host() host.Host
	StateListMiners(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error)
	StateMinerInfo(ctx context.Context, addr address.Address, tsk types.TipSetKey) (lapi.MinerInfo, error)
}

func NewTask(api API) *Task {
	return &Task{api: api}
}

type Task struct {
	api API
}

func (t *Task) Process(ctx context.Context) (model.Persistable, error) {
	miners, err := t.api.StateListMiners(ctx, types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("listing miners: %w", err)
	}

	start := time.Now()
	out := make(observed.MinerProtocolList, 0, len(miners))
	// TODO make pool size configurable via env var, optionally results channel buffer size.
	results := make(chan *observed.MinerProtocol, 50)
	pool := workerpool.New(50)

	for _, miner := range miners {
		select {
		case <-ctx.Done():
			pool.Stop()
			return nil, ctx.Err()
		default:
		}
		miner := miner
		// find the miner, if DNE abort as this indicates an error in the API as a miner was returned from StateListMiners that DNE in state tree.
		// passing EmptyTSK causes API to use current chain head.
		minerInfo, err := t.api.StateMinerInfo(ctx, miner, types.EmptyTSK)
		if err != nil {
			return nil, fmt.Errorf("getting miner %s info: %w", miner, err)
		}

		// TODO here is where a check could be performed on the miners power
		// - fetch the power actor table once outside this loop, then inspect directly in here before calling fetchMinerProtocolModel
		// - don't call StateMinerPower as it reloads the power table each time, which is pretty slow.

		pool.Submit(func() {
			// TODO make timeout configurable via env var
			fetchCtx, cancel := context.WithTimeout(ctx, time.Second*30)
			defer cancel()
			fetchMinerProtocolModel(fetchCtx, t.api, miner, minerInfo, start, results)
		})
	}

	// wait for all workers to complete then close the results channel
	go func() {
		pool.StopWait()
		close(results)
	}()

	// drain results until closed.
	for res := range results {
		// only persist miners we received complete data for.
		if res.Reachable {
			if res.Error != "" {
				log.Debugw("miner reachable but encountered error", "address", res.MinerID, "error", res.Error)
			} else {
				out = append(out, res)
			}
		} else {
			log.Debugw("miner not reachable", "address", res.MinerID, "error", res.Error)
		}
	}
	return out, nil
}

func (t *Task) Close() error {
	return nil
}

func fetchMinerProtocolModel(ctx context.Context, api API, addr address.Address, minerInfo lapi.MinerInfo, start time.Time, results chan *observed.MinerProtocol) {
	// since miners may choose if their peerID is set in their info
	var peerID string
	if minerInfo.PeerId != nil {
		peerID = minerInfo.PeerId.String()
	}

	// create the model we will export to storage
	minerModel := &observed.MinerProtocol{
		ObservedAt: start,
		MinerID:    addr.String(),
		PeerID:     peerID,
	}

	// send results regardless of below failure cases.
	defer func() {
		results <- minerModel
	}()

	// extract any multiaddresses the miner has set in their info, they may have none bail if that is the case.
	minerPeerInfo, err := getMinerAddrInfo(minerInfo)
	if err != nil {
		minerModel.Reachable = false
		minerModel.Error = err.Error()
		return
	}

	// attempt to connect to miner
	if err := api.Host().Connect(ctx, *minerPeerInfo); err != nil {
		minerModel.Reachable = false
		minerModel.Error = fmt.Errorf("failed to connect to miner: %w", err).Error()
		return
	}

	// we connected to the miner, they are reachable by the multi-address's advertised in their info
	minerModel.Reachable = true

	// get protocols supported by miner
	protos, err := api.Host().Peerstore().GetProtocols(minerPeerInfo.ID)
	if err != nil {
		minerModel.Error = fmt.Errorf("failed to get protocols for miner: %w", err).Error()
		return
	}

	// yay, miners has protocols, store it.
	minerModel.Protocols = protos

	// find miners agent version
	agentVersionI, err := api.Host().Peerstore().Get(minerPeerInfo.ID, "AgentVersion")
	if err != nil {
		minerModel.Error = fmt.Errorf("gettting agent version for miner: %w", err).Error()
		return
	}

	// alright we found everything, done.
	minerModel.Agent = agentVersionI.(string)
}

func getMinerAddrInfo(info lapi.MinerInfo) (*peer.AddrInfo, error) {
	var maddrs []multiaddr.Multiaddr
	for _, m := range info.Multiaddrs {
		ma, err := multiaddr.NewMultiaddrBytes(m)
		if err != nil {
			return nil, fmt.Errorf("miner had invalid multiaddrs in their info: %w", err)
		}
		maddrs = append(maddrs, ma)
	}
	if len(maddrs) == 0 {
		return nil, fmt.Errorf("miner has no multiaddrs set on-chain")
	}
	return &peer.AddrInfo{
		ID:    *info.PeerId,
		Addrs: maddrs,
	}, nil
}

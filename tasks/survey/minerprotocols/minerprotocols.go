package minerprotocols

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/filecoin-project/go-address"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/lily/model"
	observed "github.com/filecoin-project/lily/model/surveyed"
)

var (
	resultBufferEnv  = "LILY_SURVEY_MINER_PROTOCOL_BUFFER"
	resultBufferSize = 50

	workerPoolSizeEnv = "LILY_SURVEY_MINER_PROTOCOL_WORKERS"
	workerPoolSize    = 50

	fetchTimeoutEnv = "LILY_SURVEY_MINER_PROTOCOL_TIMEOUT_SECONDS"
	fetchTimeout    = 30
)

func init() {
	if s := os.Getenv(resultBufferEnv); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			resultBufferSize = int(v)
		}
	}
	if s := os.Getenv(workerPoolSizeEnv); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			workerPoolSize = int(v)
		}
	}
	if s := os.Getenv(fetchTimeoutEnv); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			fetchTimeout = int(v)
		}
	}
}

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
	results := make(chan *observed.MinerProtocol, resultBufferSize)
	pool := workerpool.New(workerPoolSize)

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

		pool.Submit(func() {
			fetchCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(fetchTimeout))
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
		out = append(out, res)
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

	// extract any multiaddresses the miner has set in their info, they may have none bail if that is the case.
	minerPeerInfo, err := getMinerAddrInfo(minerInfo)
	if err != nil {
		return
	}

	// attempt to connect to miner
	if err := api.Host().Connect(ctx, *minerPeerInfo); err != nil {
		return
	}

	// get protocols supported by miner
	protos, err := api.Host().Peerstore().GetProtocols(minerPeerInfo.ID)
	if err != nil {
		return
	}

	// find miners agent version
	agentVersionI, err := api.Host().Peerstore().Get(minerPeerInfo.ID, "AgentVersion")
	if err != nil {
		return
	}

	// create the model we will export to storage
	results <- &observed.MinerProtocol{
		ObservedAt: start,
		MinerID:    addr.String(),
		PeerID:     peerID,
		Agent:      agentVersionI.(string),
		Protocols:  protos,
	}

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

package processor

import (
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"

	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/services/indexer"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/genesis"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/miner"
)

const (
	GenesisTaskName = "process_genesis"
	GensisPoolName  = "genesis_tasks"

	MinerTaskName = "process_miner"
	MinerPoolName = "miner_actor_tasks"
)

// Make a redis pool
var redisPool = &redis.Pool{
	MaxActive: 64,
	MaxIdle:   64,
	Wait:      true,
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	},
}

func NewScheduler(node lapi.FullNode, pubCh chan<- model.Persistable) *Scheduler {
	genesisPool, genesisQueue := genesis.Setup(1, GenesisTaskName, GensisPoolName, redisPool, node, pubCh)
	minerPool, minerQueue := miner.Setup(64, MinerTaskName, MinerPoolName, redisPool, node, pubCh)

	pools := []*work.WorkerPool{genesisPool, minerPool}
	queues := map[string]*work.Enqueuer{
		GenesisTaskName: genesisQueue,
		MinerTaskName:   minerQueue,
	}

	return &Scheduler{
		pools:  pools,
		queues: queues,
	}
}

type Scheduler struct {
	pools  []*work.WorkerPool
	queues map[string]*work.Enqueuer
}

func (s *Scheduler) Start() {
	for _, pool := range s.pools {
		pool.Start()
	}
}

func (s *Scheduler) Stop() {
	for _, pool := range s.pools {
		// pool.Drain()
		// TODO wait for pools to drain before stopping them, will require coordination with the
		// processor such that no more tasks are added else Drain will never return.
		pool.Stop()
	}
}

func (s *Scheduler) Dispatch(tips indexer.ActorTips) error {
	for _, actors := range tips {
		for _, actor := range actors {
			switch actor.Actor.Code {
			case builtin.StorageMinerActorCodeID:
				_, err := s.queueMinerTask(actor)
				if err != nil {
					return err
				}
			case builtin.StorageMarketActorCodeID:
				//process market actor
			}
		}
	}
	return nil
}

func (s *Scheduler) queueMinerTask(info indexer.ActorInfo) (*work.Job, error) {
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ptsB, err := info.ParentTipset.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return s.queues[MinerTaskName].Enqueue(MinerTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"address":   info.Address.String(),
		"stateroot": info.ParentStateRoot.String(),
	})
}

func (s *Scheduler) queueGenesisTask(genesisTs types.TipSetKey, genesisRoot cid.Cid) (*work.Job, error) {
	tsB, err := genesisTs.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return s.queues[GenesisTaskName].EnqueueUnique(GenesisTaskName, work.Q{
		"ts":        string(tsB),
		"stateroot": genesisRoot.String(),
	})
}

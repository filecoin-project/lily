package processor

import (
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"

	"github.com/filecoin-project/sentinel-visor/services/indexer"
)

const (
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

func NewScheduler(node lapi.FullNode, publisher *Publisher) *Scheduler {
	minerPool, minerQueue := miner.Setup(64, MinerTaskName, MinerPoolName, redisPool, node, publisher.Publish)

	pools := []*work.WorkerPool{minerPool}
	queues := map[string]*work.Enqueuer{
		MinerTaskName: minerQueue,
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
				if _, err := s.queueMinerTask(actor); err != nil {
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

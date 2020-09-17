package processor

import (
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/common"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	"strconv"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/services/indexer"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/genesis"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/market"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/message"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/miner"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/power"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/reward"
)

const (
	GenesisTaskName = "process_genesis"
	GensisPoolName  = "genesis_tasks"

	MinerTaskName = "process_miner"
	MinerPoolName = "miner_actor_tasks"

	MarketTaskName = "process_market"
	MarketPoolName = "market_actor_tasks"

	MessageTaskName = "process_message"
	MessagePoolName = "message_tasks"

	PowerTaskName = "process_power"
	PowerPoolName = "power_actor_tasks"

	RewardTaskName = "process_reward"
	RewardPoolName = "reward_tasks"

	CommonTaskName = "process_common"
	CommonPoolName = "common_actor_tasks"
)

// Make a redis pool
var redisPool = &redis.Pool{
	MaxActive: 128,
	MaxIdle:   128,
	Wait:      true,
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	},
}

func NewScheduler(node lens.API, pubCh chan<- model.Persistable) *Scheduler {
	genesisPool, genesisQueue := genesis.Setup(1, GenesisTaskName, GensisPoolName, redisPool, node, pubCh)
	minerPool, minerQueue := miner.Setup(64, MinerTaskName, MinerPoolName, redisPool, node, pubCh)
	marketPool, marketQueue := market.Setup(64, MarketTaskName, MarketPoolName, redisPool, node, pubCh)
	msgPool, msgQueue := message.Setup(64, MessageTaskName, MessagePoolName, redisPool, node, pubCh)
	powerPool, powerQueue := power.Setup(64, PowerTaskName, PowerPoolName, redisPool, node, pubCh)
	rwdPool, rwdQueue := reward.Setup(4, RewardTaskName, RewardPoolName, redisPool, node, pubCh)
	comPool, comQueue := common.Setup(64, CommonTaskName, CommonPoolName, redisPool, node, pubCh)

	pools := []*work.WorkerPool{genesisPool, minerPool, marketPool, powerPool, msgPool, rwdPool, comPool}

	queues := map[string]*work.Enqueuer{
		GenesisTaskName: genesisQueue,
		MinerTaskName:   minerQueue,
		MarketTaskName:  marketQueue,
		MessageTaskName: msgQueue,
		PowerTaskName:   powerQueue,
		RewardTaskName:  rwdQueue,
		CommonTaskName:  comQueue,
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
	for ts, actors := range tips {
		// TODO this is pretty ugly, need some local cache of tsKey to ts in here somewhere.
		_, err := s.queueMessageTask(ts, actors[0].ParentStateRoot)
		if err != nil {
			return err
		}
		for _, actor := range actors {
			_, err := s.queueCommonTask(actor)
			if err != nil {
				return err
			}
			switch actor.Actor.Code {
			case builtin.StorageMinerActorCodeID:
				_, err := s.queueMinerTask(actor)
				if err != nil {
					return err
				}
			case builtin.StorageMarketActorCodeID:
				_, err := s.queueMarketTask(actor)
				if err != nil {
					return err
				}
			case builtin.StoragePowerActorCodeID:
				_, err := s.queuePowerTask(actor)
				if err != nil {
					return err
				}
			case builtin.RewardActorCodeID:
				_, err := s.queueRewardTask(actor)
				if err != nil {
					return err
				}
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
	ptsB, err := info.ParentTipSet.MarshalJSON()
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

func (s *Scheduler) queuePowerTask(info indexer.ActorInfo) (*work.Job, error) {
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return s.queues[PowerTaskName].Enqueue(PowerTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"address":   info.Address.String(),
		"stateroot": info.ParentStateRoot.String(),
	})

}

func (s *Scheduler) queueMarketTask(info indexer.ActorInfo) (*work.Job, error) {
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return s.queues[MarketTaskName].Enqueue(MarketTaskName, work.Q{
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

func (s *Scheduler) queueMessageTask(ts types.TipSetKey, st cid.Cid) (*work.Job, error) {
	tsB, err := ts.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return s.queues[MessageTaskName].Enqueue(MessageTaskName, work.Q{
		"ts":        string(tsB),
		"stateroot": st.String(),
	})
}

func (s *Scheduler) queueRewardTask(info indexer.ActorInfo) (*work.Job, error) {
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return s.queues[RewardTaskName].Enqueue(RewardTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"stateroot": info.ParentStateRoot.String(),
	})
}

func (s *Scheduler) queueCommonTask(info indexer.ActorInfo) (*work.Job, error) {
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return s.queues[CommonTaskName].Enqueue(CommonTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"stateroot": info.ParentStateRoot.String(),
		"address":   info.Address.String(),
		"head":      info.Actor.Head.String(),
		"code":      info.Actor.Code.String(),
		"balance":   info.Actor.Balance.String(),
		"nonce":     strconv.FormatUint(info.Actor.Nonce, 10),
	})
}

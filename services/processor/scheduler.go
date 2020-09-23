package processor

import (
	"context"
	"os"
	"strconv"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/services/indexer"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/common"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/genesis"
	init_ "github.com/filecoin-project/sentinel-visor/services/processor/tasks/init"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/market"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/message"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/miner"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/power"
	"github.com/filecoin-project/sentinel-visor/services/processor/tasks/reward"
	"github.com/filecoin-project/sentinel-visor/tasks"
)

var log = logging.Logger("scheduler")

const (
	EnvRedisMaxActive = "VISOR_REDIS_MAX_ACTIVE"
	EnvRedisMaxIdle   = "VISOR_REDIS_MAX_IDLE"
	EnvRedisNetwork   = "VISOR_REDIS_NETWORK"
	EnvRedisAddress   = "VISOR_REDIS_ADDRESS"
)

var (
	RedisMaxActive int64
	RedisMaxIdle   int64
	RedisNetwork   string
	RedisAddress   string
)

func init() {
	RedisMaxActive = 128
	RedisMaxIdle = 128
	RedisNetwork = "tcp"
	RedisAddress = ":6379"

	if maxActiveStr := os.Getenv(EnvRedisMaxActive); maxActiveStr != "" {
		max, err := strconv.ParseInt(maxActiveStr, 10, 64)
		if err != nil {
			log.Errorw("setting redis max active", "error", err)
		} else {
			RedisMaxActive = max
		}
	}

	if maxIdlStr := os.Getenv(EnvRedisMaxIdle); maxIdlStr != "" {
		max, err := strconv.ParseInt(maxIdlStr, 10, 64)
		if err != nil {
			log.Errorw("setting redis max idle", "error", err)
		} else {
			RedisMaxIdle = max
		}
	}

	if network := os.Getenv(EnvRedisNetwork); network != "" {
		RedisNetwork = network
	}

	if address := os.Getenv(EnvRedisAddress); address != "" {
		RedisAddress = address
	}
}



func NewScheduler(node lens.API, pubCh chan<- model.Persistable) *Scheduler {
	// Make a redis pool
	var redisPool = &redis.Pool{
		MaxActive: int(RedisMaxActive),
		MaxIdle:   int(RedisMaxIdle),
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial(RedisNetwork, RedisAddress)
		},
	}

	genesisPool, genesisQueue := genesis.Setup(1, tasks.GenesisTaskName, tasks.GenesisPoolName, redisPool, node, pubCh)
	minerPool, minerQueue := miner.Setup(64, tasks.MinerTaskName, tasks.MinerPoolName, redisPool, node, pubCh)
	marketPool, marketQueue := market.Setup(64, tasks.MarketTaskName, tasks.MarketPoolName, redisPool, node, pubCh)
	msgPool, msgQueue := message.Setup(64, tasks.MessageTaskName, tasks.MessagePoolName, redisPool, node, pubCh)
	powerPool, powerQueue := power.Setup(64, tasks.PowerTaskName, tasks.PowerPoolName, redisPool, node, pubCh)
	rwdPool, rwdQueue := reward.Setup(4, tasks.RewardTaskName, tasks.RewardPoolName, redisPool, node, pubCh)
	comPool, comQueue := common.Setup(64, tasks.CommonTaskName, tasks.CommonPoolName, redisPool, node, pubCh)
	initPool, initQueue := init_.Setup(64, tasks.InitActorTaskName, tasks.InitActorPoolName, redisPool, node, pubCh)

	pools := []*work.WorkerPool{genesisPool, minerPool, marketPool, powerPool, msgPool, rwdPool, comPool, initPool}

	queues := map[string]*work.Enqueuer{
		tasks.GenesisTaskName:   genesisQueue,
		tasks.MinerTaskName:     minerQueue,
		tasks.MarketTaskName:    marketQueue,
		tasks.MessageTaskName:   msgQueue,
		tasks.PowerTaskName:     powerQueue,
		tasks.RewardTaskName:    rwdQueue,
		tasks.CommonTaskName:    comQueue,
		tasks.InitActorTaskName: initQueue,
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
			return xerrors.Errorf("queue message task: %w", err)
		}
		for _, actor := range actors {
			_, err := s.queueCommonTask(actor)
			if err != nil {
				return xerrors.Errorf("queue common task: %w", err)
			}
			switch actor.Actor.Code {
			case builtin.InitActorCodeID:
				_, err := s.queueInitTask(actor)
				if err != nil {
					return xerrors.Errorf("queue init task: %w", err)
				}
			case builtin.StorageMinerActorCodeID:
				_, err := s.queueMinerTask(actor)
				if err != nil {
					return xerrors.Errorf("queue miner task: %w", err)
				}
			case builtin.StorageMarketActorCodeID:
				_, err := s.queueMarketTask(actor)
				if err != nil {
					return xerrors.Errorf("queue market task: %w", err)
				}
			case builtin.StoragePowerActorCodeID:
				_, err := s.queuePowerTask(actor)
				if err != nil {
					return xerrors.Errorf("queue power task: %w", err)
				}
			case builtin.RewardActorCodeID:
				_, err := s.queueRewardTask(actor)
				if err != nil {
					return xerrors.Errorf("queue reward task: %w", err)
				}
			}
		}
	}
	return nil
}

func (s *Scheduler) queueMinerTask(info indexer.ActorInfo) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.MinerPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal parent tipset key: %w", err)
	}
	return s.queues[tasks.MinerTaskName].Enqueue(tasks.MinerTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"address":   info.Address.String(),
		"stateroot": info.ParentStateRoot.String(),
	})
}

func (s *Scheduler) queuePowerTask(info indexer.ActorInfo) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.PowerPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal parent tipset key: %w", err)
	}
	return s.queues[tasks.PowerTaskName].Enqueue(tasks.PowerTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"address":   info.Address.String(),
		"stateroot": info.ParentStateRoot.String(),
	})

}

func (s *Scheduler) queueMarketTask(info indexer.ActorInfo) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.MarketPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal parent tipset key: %w", err)
	}
	return s.queues[tasks.MarketTaskName].Enqueue(tasks.MarketTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"address":   info.Address.String(),
		"stateroot": info.ParentStateRoot.String(),
	})
}

func (s *Scheduler) queueInitTask(info indexer.ActorInfo) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.InitActorPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal parent tipset key: %w", err)
	}
	return s.queues[tasks.InitActorTaskName].Enqueue(tasks.InitActorTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"address":   info.Address.String(),
		"stateroot": info.ParentStateRoot.String(),
	})

}

func (s *Scheduler) queueGenesisTask(genesisTs types.TipSetKey, genesisRoot cid.Cid) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.GenesisPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := genesisTs.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	return s.queues[tasks.GenesisTaskName].EnqueueUnique(tasks.GenesisTaskName, work.Q{
		"ts":        string(tsB),
		"stateroot": genesisRoot.String(),
	})
}

func (s *Scheduler) queueMessageTask(ts types.TipSetKey, st cid.Cid) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.MessagePoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := ts.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	return s.queues[tasks.MessageTaskName].Enqueue(tasks.MessageTaskName, work.Q{
		"ts":        string(tsB),
		"stateroot": st.String(),
	})
}

func (s *Scheduler) queueRewardTask(info indexer.ActorInfo) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.RewardPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return s.queues[tasks.RewardTaskName].Enqueue(tasks.RewardTaskName, work.Q{
		"ts":        string(tsB),
		"pts":       string(ptsB),
		"head":      info.Actor.Head.String(),
		"stateroot": info.ParentStateRoot.String(),
	})
}

func (s *Scheduler) queueCommonTask(info indexer.ActorInfo) (*work.Job, error) {
	ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.TaskNS, tasks.CommonPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(1))
	tsB, err := info.TipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal tipset key: %w", err)
	}
	ptsB, err := info.ParentTipSet.MarshalJSON()
	if err != nil {
		return nil, xerrors.Errorf("marshal parent tipset key: %w", err)
	}

	return s.queues[tasks.CommonTaskName].Enqueue(tasks.CommonTaskName, work.Q{
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

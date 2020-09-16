package miner

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/events/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	api "github.com/filecoin-project/sentinel-visor/lens/lotus"
	"github.com/filecoin-project/sentinel-visor/model"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node api.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessMinerTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessMinerTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("minertask")
		return next()
	})
	logging.SetLogLevel("minertask", "info")
	// log all task
	pool.Middleware((*ProcessMinerTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessMinerTask).Task)

	return pool, queue
}

type ProcessMinerTask struct {
	node lapi.FullNode
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	maddr     address.Address
	head      cid.Cid
	tsKey     types.TipSetKey
	ptsKey    types.TipSetKey
	stateroot cid.Cid
}

func (mac *ProcessMinerTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	mac.log.Infow("Starting Miner Task", "name", job.Name, "Args", job.Args)
	return next()
}

func (mac *ProcessMinerTask) ParseArgs(job *work.Job) error {
	addrStr := job.ArgString("address")
	if err := job.ArgError(); err != nil {
		return err
	}

	headStr := job.ArgString("head")
	if err := job.ArgError(); err != nil {
		return err
	}

	srStr := job.ArgString("stateroot")
	if err := job.ArgError(); err != nil {
		return err
	}

	tsStr := job.ArgString("ts")
	if err := job.ArgError(); err != nil {
		return err
	}

	ptsStr := job.ArgString("pts")
	if err := job.ArgError(); err != nil {
		return err
	}

	maddr, err := address.NewFromString(addrStr)
	if err != nil {
		return err
	}

	mhead, err := cid.Decode(headStr)
	if err != nil {
		return err
	}

	mstateroot, err := cid.Decode(srStr)
	if err != nil {
		return err
	}

	var tsKey types.TipSetKey
	if err := tsKey.UnmarshalJSON([]byte(tsStr)); err != nil {
		return err
	}

	var ptsKey types.TipSetKey
	if err := ptsKey.UnmarshalJSON([]byte(ptsStr)); err != nil {
		return err
	}

	mac.maddr = maddr
	mac.head = mhead
	mac.tsKey = tsKey
	mac.ptsKey = ptsKey
	mac.stateroot = mstateroot
	return nil
}

func (mac *ProcessMinerTask) Task(job *work.Job) error {
	if err := mac.ParseArgs(job); err != nil {
		return err
	}
	// TODO:
	// - all processing below can and probably should be done in parallel.
	// - processing is incomplete, see below TODO about sector inspection.
	// - need caching infront of the lotus api to avoid refetching power for same tipset.
	ctx := context.TODO()
	store := api.NewAPIIpldStore(ctx, mac.node)

	// generic actor state of the miner.
	mactor, err := mac.node.StateGetActor(ctx, mac.maddr, mac.tsKey)
	if err != nil {
		return err
	}

	// actual miner actor state and miner info
	var mstate miner.State
	astb, err := mac.node.ChainReadObj(ctx, mactor.Head)
	if err != nil {
		return err
	}
	if err := mstate.UnmarshalCBOR(bytes.NewReader(astb)); err != nil {
		return err
	}
	minfo, err := mstate.GetInfo(store)
	if err != nil {
		return err
	}

	// miner raw and qual power
	// TODO this needs caching so we don't re-fetch the power actors claim table for every tipset.
	minerPower, err := mac.node.StateMinerPower(ctx, mac.maddr, mac.tsKey)
	if err != nil {
		return err
	}

	// miner precommits added and removed
	preCommitChanges, err := minerPreCommitChanges(ctx, mac.node, mac.maddr, mac.tsKey, mac.ptsKey)
	if err != nil {
		return fmt.Errorf("precommit changes: %v", err)
	}

	// miner sectors added, removed, and extended
	sectorChanges, err := minerSectorChanges(ctx, mac.node, mac.maddr, mac.tsKey, mac.ptsKey)
	if err != nil {
		return fmt.Errorf("sector changes: %v", err)
	}

	// miner partition changes
	partitionsDiff, err := minerPartitionsDiff(ctx, mac.node, mac.maddr, mac.tsKey, mac.ptsKey)
	if err != nil {
		return fmt.Errorf("partition diff: %v", err)
	}

	// TODO we still need to do a little bit more processing here around sectors to get all the info we need, but this is okay for spike.

	mac.pubCh <- &minermodel.MinerTaskResult{
		Ts:               mac.tsKey,
		Pts:              mac.ptsKey,
		Addr:             mac.maddr,
		StateRoot:        mac.stateroot,
		Actor:            mactor,
		State:            mstate,
		Info:             minfo,
		Power:            minerPower,
		PreCommitChanges: preCommitChanges,
		SectorChanges:    sectorChanges,
		PartitionDiff:    partitionsDiff,
	}
	return nil
}

func minerPreCommitChanges(ctx context.Context, node api.API, maddr address.Address, ts, pts types.TipSetKey) (*state.MinerPreCommitChanges, error) {
	pred := state.NewStatePredicates(node)
	changed, val, err := pred.OnMinerActorChange(maddr, pred.OnMinerPreCommitChange())(ctx, pts, ts)
	if err != nil {
		return nil, xerrors.Errorf("Failed to diff miner precommit amt: %w", err)
	}
	if !changed {
		return nil, nil
	}
	out := val.(*state.MinerPreCommitChanges)
	return out, nil
}

func minerSectorChanges(ctx context.Context, node api.API, maddr address.Address, ts, pts types.TipSetKey) (*state.MinerSectorChanges, error) {
	pred := state.NewStatePredicates(node)
	changed, val, err := pred.OnMinerActorChange(maddr, pred.OnMinerSectorChange())(ctx, pts, ts)
	if err != nil {
		return nil, xerrors.Errorf("Failed to diff miner sectors amt: %w", err)
	}
	if !changed {
		return nil, nil
	}
	out := val.(*state.MinerSectorChanges)
	return out, nil
}

func minerPartitionsDiff(ctx context.Context, node api.API, maddr address.Address, ts, pts types.TipSetKey) (map[uint64]*minermodel.PartitionStatus, error) {
	store := api.NewAPIIpldStore(ctx, node)

	curMiner, err := minerStateAt(ctx, node, maddr, ts)
	if err != nil {
		return nil, err
	}

	prevMiner, err := minerStateAt(ctx, node, maddr, pts)
	if err != nil {
		return nil, err
	}
	dlIdx := prevMiner.CurrentDeadline

	//
	// load the prev deadline and partitions
	//
	prevDls, err := prevMiner.LoadDeadlines(store)
	if err != nil {
		return nil, err
	}
	var prevDl miner.Deadline
	if err := store.Get(ctx, prevDls.Due[dlIdx], &prevDl); err != nil {
		return nil, err
	}

	prevPartitions, err := prevDl.PartitionsArray(store)
	if err != nil {
		return nil, err
	}

	//
	// load the cur deadline and partitions
	//
	curDls, err := curMiner.LoadDeadlines(store)
	if err != nil {
		return nil, err
	}

	var curDl miner.Deadline
	if err := store.Get(ctx, curDls.Due[dlIdx], &curDl); err != nil {
		return nil, err
	}

	curPartitions, err := curDl.PartitionsArray(store)
	if err != nil {
		return nil, err
	}

	//
	// walk all miner partitions and calculate their differences
	//
	out := make(map[uint64]*minermodel.PartitionStatus)
	// TODO this can be optimized by inspecting the miner state for partitions that have changed and only inspecting those.
	// FIXME: account for curPartition array having partitions not found in prevPartition array.
	var prevPart miner.Partition
	if err := prevPartitions.ForEach(&prevPart, func(i int64) error {
		var curPart miner.Partition
		if found, err := curPartitions.Get(uint64(i), &curPart); err != nil {
			return err
		} else if !found {
			panic("Undefined behaviour when a partition is removed.")
		}
		partitionDiff, err := diffPartition(store, prevPart, curPart)
		if err != nil {
			return err
		}
		out[uint64(i)] = partitionDiff

		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func diffPartition(store *api.APIIpldStore, prevPart, curPart miner.Partition) (*minermodel.PartitionStatus, error) {
	// all the sectors that were in previous but not in current
	allRemovedSectors, err := bitfield.SubtractBitField(prevPart.Sectors, curPart.Sectors)
	if err != nil {
		return nil, err
	}

	// list of sectors that were terminated before their expiration.
	terminatedEarlyArr, err := adt.AsArray(store, curPart.EarlyTerminated)
	if err != nil {
		return nil, err
	}

	expired := bitfield.New()
	var bf bitfield.BitField
	if err := terminatedEarlyArr.ForEach(&bf, func(i int64) error {
		// expired = all removals - termination
		expirations, err := bitfield.SubtractBitField(allRemovedSectors, bf)
		if err != nil {
			return err
		}
		// merge with expired sectors from other epochs
		expired, err = bitfield.MergeBitFields(expirations, expired)
		if err != nil {
			return nil
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// terminated = all removals - expired
	terminated, err := bitfield.SubtractBitField(allRemovedSectors, expired)
	if err != nil {
		return nil, err
	}

	// faults in current but not previous
	faults, err := bitfield.SubtractBitField(curPart.Recoveries, prevPart.Recoveries)
	if err != nil {
		return nil, err
	}

	// recoveries in current but not previous
	inRecovery, err := bitfield.SubtractBitField(curPart.Recoveries, prevPart.Recoveries)
	if err != nil {
		return nil, err
	}

	// all current good sectors
	newActiveSectors, err := curPart.ActiveSectors()
	if err != nil {
		return nil, err
	}

	// sectors that were previously fault and are now currently active are considered recovered.
	recovered, err := bitfield.IntersectBitField(prevPart.Faults, newActiveSectors)
	if err != nil {
		return nil, err
	}

	return &minermodel.PartitionStatus{
		Terminated: terminated,
		Expired:    expired,
		Faulted:    faults,
		InRecovery: inRecovery,
		Recovered:  recovered,
	}, nil
}

func minerStateAt(ctx context.Context, node api.API, maddr address.Address, tskey types.TipSetKey) (miner.State, error) {
	prevActor, err := node.StateGetActor(ctx, maddr, tskey)
	if err != nil {
		return miner.State{}, err
	}
	var out miner.State
	// Get the miner state info
	astb, err := node.ChainReadObj(ctx, prevActor.Head)
	if err != nil {
		return miner.State{}, err
	}
	if err := out.UnmarshalCBOR(bytes.NewReader(astb)); err != nil {
		return miner.State{}, err
	}
	return out, nil
}

package actorstate

import (
	"bytes"
	"context"

	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
)

// was services/processor/tasks/miner/miner.go

// StorageMinerExtractor extracts miner actor state
type StorageMinerExtractor struct{}

func init() {
	Register(builtin.StorageMinerActorCodeID, StorageMinerExtractor{})
}

func (m StorageMinerExtractor) Extract(ctx context.Context, a ActorInfo, node lens.API) (model.Persistable, error) {
	// TODO:
	// - all processing below can and probably should be done in parallel.
	// - processing is incomplete, see below TODO about sector inspection.
	// - need caching infront of the lotus api to avoid refetching power for same tipset.
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	curActor, err := node.StateGetActor(ctx, a.Address, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner actor: %w", err)
	}

	curTipset, err := node.ChainGetTipSet(ctx, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current tipset: %w", err)
	}

	curState, err := miner.Load(node.Store(), curActor)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner state: %w", err)
	}

	minfo, err := curState.Info()
	if err != nil {
		return nil, xerrors.Errorf("loading miner info: %w", err)
	}

	// miner raw and qual power
	// TODO this needs caching so we don't re-fetch the power actors claim table (that holds this info) for every tipset.
	minerPower, err := node.StateMinerPower(ctx, a.Address, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading miner power: %w", err)
	}

	// needed for diffing.
	prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading previous miner actor: %w", err)
	}

	prevState, err := miner.Load(node.Store(), prevActor)
	if err != nil {
		return nil, xerrors.Errorf("loading previous miner actor state: %w", err)
	}

	preCommitChanges, err := miner.DiffPreCommits(prevState, curState)
	if err != nil {
		return nil, xerrors.Errorf("diffing miner precommits: %w", err)
	}

	sectorChanges, err := miner.DiffSectors(prevState, curState)
	if err != nil {
		return nil, xerrors.Errorf("diffing miner sectors: %w", err)
	}

	// miner partition changes
	partitionsDiff, err := m.minerPartitionsDiff(ctx, prevState, curState)
	if err != nil {
		return nil, xerrors.Errorf("diffing miner partitions: %v", err)
	}

	// miner posts
	posts, err := m.minerPosts(ctx, &a, curTipset.Height(), curState, node)
	if err != nil {
		return nil, xerrors.Errorf("constructing miner posts: %v", err)
	}

	// TODO we still need to do a little bit more processing here around sectors to get all the info we need, but this is okay for spike.

	return &minermodel.MinerTaskResult{
		Ts:               a.TipSet,
		Pts:              a.ParentTipSet,
		Addr:             a.Address,
		Height:           int64(curTipset.Height()),
		StateRoot:        a.ParentStateRoot,
		Actor:            curActor,
		State:            curState,
		Info:             minfo,
		Power:            minerPower,
		PreCommitChanges: preCommitChanges,
		SectorChanges:    sectorChanges,
		PartitionDiff:    partitionsDiff,
		Posts:            posts,
	}, nil
}

func (m StorageMinerExtractor) minerPartitionsDiff(ctx context.Context, prevState, curState miner.State) (map[uint64]*minermodel.PartitionStatus, error) {
	return nil, nil
}

func (m StorageMinerExtractor) minerPosts(ctx context.Context, actor *ActorInfo, epoch abi.ChainEpoch, curState miner.State, node lens.API) (map[uint64]cid.Cid, error) {
	posts := make(map[uint64]cid.Cid)
	block := actor.TipSet.Cids()[0]
	msgs, err := node.ChainGetBlockMessages(ctx, block)
	if err != nil {
		return nil, xerrors.Errorf("diffing miner posts: %v", err)
	}

	var partitions []miner.Partition
	loadPartitions := func(state miner.State, epoch abi.ChainEpoch) ([]miner.Partition, error) {
		info, err := state.DeadlineInfo(epoch)
		if err != nil {
			return nil, err
		}
		dline, err := state.LoadDeadline(info.Index)
		if err != nil {
			return nil, err
		}
		partitions := make([]miner.Partition, 0, 8)
		if err := dline.ForEachPartition(func(idx uint64, p miner.Partition) error {
			partitions[idx] = p
			return nil
		}); err != nil {
			return nil, err
		}
		return partitions, nil
	}

	for _, msg := range msgs.BlsMessages {
		if msg.To == actor.Address && msg.Method == 5 /* miner.SubmitWindowedPoSt */ {
			sectors := make([]uint64, 0)
			rcpt, err := node.StateGetReceipt(ctx, msg.Cid(), actor.TipSet)
			if err != nil {
				return nil, err
			}
			if rcpt.ExitCode.IsError() {
				continue
			}
			params := miner.SubmitWindowedPoStParams{}
			if err := params.UnmarshalCBOR(bytes.NewBuffer(msg.Params)); err != nil {
				return nil, err
			}

			if partitions == nil {
				partitions, err = loadPartitions(curState, epoch)
				if err != nil {
					return nil, err
				}
			}

			for _, p := range params.Partitions {
				all, err := partitions[p.Index].AllSectors()
				if err != nil {
					return nil, err
				}
				proven, err := bitfield.SubtractBitField(all, p.Skipped)
				if err != nil {
					return nil, err
				}

				proven.ForEach(func(sector uint64) error {
					sectors = append(sectors, sector)
					return nil
				})
			}

			for _, s := range sectors {
				posts[s] = msg.Cid()
			}
		}
	}
	return posts, nil
}

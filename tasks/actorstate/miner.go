package actorstate

import (
	"bytes"
	"context"

	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
)

// was services/processor/tasks/miner/miner.go

// StorageMinerExtractor extracts miner actor state
type StorageMinerExtractor struct{}

func init() {
	Register(sa0builtin.StorageMinerActorCodeID, StorageMinerExtractor{})
	Register(sa2builtin.StorageMinerActorCodeID, StorageMinerExtractor{})
}

func (m StorageMinerExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
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

	sectorEvents, err := m.computeSectorEvents(ctx, node, a, sectorChanges, preCommitChanges, partitionsDiff)
	if err != nil {
		return nil, xerrors.Errorf("collecting miner sector events")
	}

	return &minermodel.MinerTaskResult{
		Ts:               a.TipSet,
		Pts:              a.ParentTipSet,
		Addr:             a.Address,
		Height:           curTipset.Height(),
		StateRoot:        a.ParentStateRoot,
		Actor:            curActor,
		State:            curState,
		Info:             minfo,
		Power:            minerPower,
		PreCommitChanges: preCommitChanges,
		SectorChanges:    sectorChanges,
		Posts:            posts,
		SectorEvents:     sectorEvents,
	}, nil
}

func (m StorageMinerExtractor) computeSectorEvents(ctx context.Context, node ActorStateAPI, a ActorInfo, sc *miner.SectorChanges, pc *miner.PreCommitChanges, ps *minermodel.PartitionStatus) (minermodel.MinerSectorEventList, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtractor.computeSectorEvents")
	defer span.End()

	out := minermodel.MinerSectorEventList{}
	sectorAdds := make(map[abi.SectorNumber]miner.SectorOnChainInfo)

	// if there were changes made to the miners partition lists
	if ps != nil {
		// build an index of removed sector expiration's for comparison below.
		removedSectors, err := node.StateMinerSectors(ctx, a.Address, &ps.Removed, a.TipSet)
		if err != nil {
			return nil, xerrors.Errorf("fetching miners removed sectors: %w", err)
		}
		rmExpireIndex := make(map[uint64]abi.ChainEpoch)
		for _, rm := range removedSectors {
			rmExpireIndex[uint64(rm.SectorNumber)] = rm.Expiration
		}

		// track terminated and expired sectors
		if err := ps.Removed.ForEach(func(u uint64) error {
			event := minermodel.SectorTerminated
			expiration := rmExpireIndex[u]
			if expiration == a.Epoch {
				event = minermodel.SectorExpired
			}
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     event,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners removed sectors: %w", err)
		}

		// track recovering sectors
		if err := ps.Recovering.ForEach(func(u uint64) error {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     minermodel.SectorRecovering,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners recovering sectors: %w", err)
		}

		// track faulted sectors
		if err := ps.Faulted.ForEach(func(u uint64) error {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     minermodel.SectorFaulted,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners faulted sectors: %w", err)
		}

		// track recovered sectors
		if err := ps.Recovered.ForEach(func(u uint64) error {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  u,
				Event:     minermodel.SectorRecovered,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners recovered sectors: %w", err)
		}
	}

	// if there were changes made to the miners sectors list
	if sc != nil {
		// track sector add and commit-capacity add
		for _, add := range sc.Added {
			event := minermodel.SectorAdded
			if len(add.DealIDs) == 0 {
				event = minermodel.CommitCapacityAdded
			}
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  uint64(add.SectorNumber),
				Event:     event,
			})
			sectorAdds[add.SectorNumber] = add
		}

		// track sector extensions
		for _, mod := range sc.Extended {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  uint64(mod.To.SectorNumber),
				Event:     minermodel.SectorExtended,
			})
		}

	}

	// if there were changes made to the miners precommit list
	if pc != nil {
		// track precommit addition
		for _, add := range pc.Added {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Epoch),
				MinerID:   a.Address.String(),
				StateRoot: a.ParentStateRoot.String(),
				SectorID:  uint64(add.Info.SectorNumber),
				Event:     minermodel.PreCommitAdded,
			})
		}
	}

	return out, nil
}

func (m StorageMinerExtractor) minerPartitionsDiff(ctx context.Context, prevState, curState miner.State) (*minermodel.PartitionStatus, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtractor.minerPartitionDiff")
	defer span.End()

	dlDiff, err := miner.DiffDeadlines(prevState, curState)
	if err != nil {
		return nil, err
	}

	if dlDiff == nil {
		return nil, nil
	}

	removed := bitfield.New()
	faulted := bitfield.New()
	recovered := bitfield.New()
	recovering := bitfield.New()

	for _, deadline := range dlDiff {
		for _, partition := range deadline {
			removed, err = bitfield.MergeBitFields(removed, partition.Removed)
			if err != nil {
				return nil, err
			}
			faulted, err = bitfield.MergeBitFields(faulted, partition.Faulted)
			if err != nil {
				return nil, err
			}
			recovered, err = bitfield.MergeBitFields(recovered, partition.Recovered)
			if err != nil {
				return nil, err
			}
			recovering, err = bitfield.MergeBitFields(recovering, partition.Recovering)
			if err != nil {
				return nil, err
			}
		}
	}
	return &minermodel.PartitionStatus{
		Removed:    removed,
		Faulted:    faulted,
		Recovering: recovering,
		Recovered:  recovered,
	}, nil
}

func (m StorageMinerExtractor) minerPosts(ctx context.Context, actor *ActorInfo, epoch abi.ChainEpoch, curState miner.State, node ActorStateAPI) (map[uint64]cid.Cid, error) {
	posts := make(map[uint64]cid.Cid)
	block := actor.TipSet.Cids()[0]
	msgs, err := node.ChainGetBlockMessages(ctx, block)
	if err != nil {
		return nil, xerrors.Errorf("diffing miner posts: %v", err)
	}

	var partitions map[uint64]miner.Partition
	loadPartitions := func(state miner.State, epoch abi.ChainEpoch) (map[uint64]miner.Partition, error) {
		info, err := state.DeadlineInfo(epoch)
		if err != nil {
			return nil, err
		}
		dline, err := state.LoadDeadline(info.Index)
		if err != nil {
			return nil, err
		}
		pmap := make(map[uint64]miner.Partition)
		if err := dline.ForEachPartition(func(idx uint64, p miner.Partition) error {
			pmap[idx] = p
			return nil
		}); err != nil {
			return nil, err
		}
		return pmap, nil
	}

	processPostMsg := func(msg *types.Message) error {
		sectors := make([]uint64, 0)
		rcpt, err := node.StateGetReceipt(ctx, msg.Cid(), actor.TipSet)
		if err != nil {
			return err
		}
		if rcpt == nil || rcpt.ExitCode.IsError() {
			return nil
		}
		params := miner.SubmitWindowedPoStParams{}
		if err := params.UnmarshalCBOR(bytes.NewBuffer(msg.Params)); err != nil {
			return err
		}

		if partitions == nil {
			partitions, err = loadPartitions(curState, epoch)
			if err != nil {
				return err
			}
		}

		for _, p := range params.Partitions {
			all, err := partitions[p.Index].AllSectors()
			if err != nil {
				return err
			}
			proven, err := bitfield.SubtractBitField(all, p.Skipped)
			if err != nil {
				return err
			}

			proven.ForEach(func(sector uint64) error {
				sectors = append(sectors, sector)
				return nil
			})
		}

		for _, s := range sectors {
			posts[s] = msg.Cid()
		}
		return nil
	}

	for _, msg := range msgs.BlsMessages {
		if msg.To == actor.Address && msg.Method == 5 /* miner.SubmitWindowedPoSt */ {
			if err := processPostMsg(msg); err != nil {
				return nil, err
			}
		}
	}
	for _, msg := range msgs.SecpkMessages {
		if msg.Message.To == actor.Address && msg.Message.Method == 5 /* miner.SubmitWindowedPoSt */ {
			if err := processPostMsg(&msg.Message); err != nil {
				return nil, err
			}
		}
	}
	return posts, nil
}

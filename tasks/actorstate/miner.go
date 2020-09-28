package actorstate

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
)

// was services/processor/tasks/miner/miner.go

// StorageMinerExtracter extracts miner actor state
type StorageMinerExtracter struct{}

func init() {
	Register(builtin.StorageMinerActorCodeID, StorageMinerExtracter{})
}

func (m StorageMinerExtracter) Extract(ctx context.Context, a ActorInfo, node lens.API) (model.Persistable, error) {
	// TODO:
	// - all processing below can and probably should be done in parallel.
	// - processing is incomplete, see below TODO about sector inspection.
	// - need caching infront of the lotus api to avoid refetching power for same tipset.
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtracter")
	defer span.End()

	curActor, err := node.StateGetActor(ctx, a.Address, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner actor: %w", err)
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

	// TODO we still need to do a little bit more processing here around sectors to get all the info we need, but this is okay for spike.

	return &minermodel.MinerTaskResult{
		Ts:               a.TipSet,
		Pts:              a.ParentTipSet,
		Addr:             a.Address,
		StateRoot:        a.ParentStateRoot,
		Actor:            curActor,
		State:            curState,
		Info:             minfo,
		Power:            minerPower,
		PreCommitChanges: preCommitChanges,
		SectorChanges:    sectorChanges,
		PartitionDiff:    partitionsDiff,
	}, nil
}

func (m StorageMinerExtracter) minerPartitionsDiff(ctx context.Context, prevState, curState miner.State) (map[uint64]*minermodel.PartitionStatus, error) {
	return nil, nil
}

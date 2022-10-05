package precommitevent

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	minertypes "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lily/tasks/actorstate/miner/extraction"
)

var log = logging.Logger("precommitevent")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&PreCommitEvent{}, Extract)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&PreCommitEvent{}, supportedActors)
}

var _ v2.LilyModel = (*PreCommitEvent)(nil)

type PrecommitEventType int64

const (
	PreCommitAdded PrecommitEventType = iota
	PreCommitRemoved
)

func (e PrecommitEventType) String() string {
	switch e {
	case PreCommitAdded:
		return "PRECOMMIT_ADDED"
	case PreCommitRemoved:
		return "PRECOMMIT_REMOVED"
	}
	panic(fmt.Sprintf("unhanded type %d developer error", e))
}

type SectorPreCommitInfo struct {
	SealProof              abi.RegisteredSealProof
	SectorNumber           abi.SectorNumber
	SealedCID              cid.Cid
	SealRandEpoch          abi.ChainEpoch
	DealIDs                []abi.DealID
	Expiration             abi.ChainEpoch
	ReplaceCapacity        bool
	ReplaceSectorDeadline  uint64
	ReplaceSectorPartition uint64
	ReplaceSectorNumber    abi.SectorNumber
}

type SectorPreCommitOnChainInfo struct {
	Info               SectorPreCommitInfo
	PreCommitDeposit   abi.TokenAmount
	PreCommitEpoch     abi.ChainEpoch
	DealWeight         abi.DealWeight
	VerifiedDealWeight abi.DealWeight
}

type PreCommitEvent struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	Miner     address.Address
	Event     PrecommitEventType
	Precommit SectorPreCommitOnChainInfo
}

func (p *PreCommitEvent) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(PreCommitEvent{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (p *PreCommitEvent) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    p.Height,
		StateRoot: p.StateRoot,
	}
}

func (t *PreCommitEvent) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *PreCommitEvent) ToStorageBlock() (block.Block, error) {
	data, err := t.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (t *PreCommitEvent) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func Extract(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	extState, err := extraction.LoadMinerStates(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	var preCommitChanges = miner.MakePreCommitChanges()
	if extState.ParentState() == nil {
		// If the miner doesn't have previous state list all of its current precommits
		if err = extState.CurrentState().ForEachPrecommittedSector(func(info minertypes.SectorPreCommitOnChainInfo) error {
			preCommitChanges.Added = append(preCommitChanges.Added, info)
			return nil
		}); err != nil {
			return nil, err
		}

	} else {
		// If the miner has previous state compute the list of new sectors and precommit in its current state.
		start := time.Now()
		// collect changes made to miner precommit map (HAMT)
		preCommitChanges, err = api.DiffPreCommits(ctx, a.Address, a.Current, a.Executed, extState.ParentState(), extState.CurrentState())
		if err != nil {
			return nil, fmt.Errorf("diffing precommits %w", err)
		}
		log.Debugw("diff precommits", "miner", a.Address, "duration", time.Since(start))
	}

	idx := 0
	out := make([]v2.LilyModel, len(preCommitChanges.Added)+len(preCommitChanges.Removed))
	for _, change := range preCommitChanges.Added {
		out[idx] = &PreCommitEvent{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Miner:     a.Address,
			Event:     PreCommitAdded,
			Precommit: SectorPreCommitOnChainInfo{
				Info: SectorPreCommitInfo{
					SealProof:              change.Info.SealProof,
					SectorNumber:           change.Info.SectorNumber,
					SealedCID:              change.Info.SealedCID,
					SealRandEpoch:          change.Info.SealRandEpoch,
					DealIDs:                change.Info.DealIDs,
					Expiration:             change.Info.Expiration,
					ReplaceCapacity:        change.Info.ReplaceCapacity,
					ReplaceSectorDeadline:  change.Info.ReplaceSectorDeadline,
					ReplaceSectorPartition: change.Info.ReplaceSectorPartition,
					ReplaceSectorNumber:    change.Info.ReplaceSectorNumber,
				},
				PreCommitDeposit:   change.PreCommitDeposit,
				PreCommitEpoch:     change.PreCommitEpoch,
				DealWeight:         change.DealWeight,
				VerifiedDealWeight: change.VerifiedDealWeight,
			},
		}
		idx++
	}
	for _, change := range preCommitChanges.Removed {
		out[idx] = &PreCommitEvent{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Miner:     a.Address,
			Event:     PreCommitRemoved,
			Precommit: SectorPreCommitOnChainInfo{
				Info: SectorPreCommitInfo{
					SealProof:              change.Info.SealProof,
					SectorNumber:           change.Info.SectorNumber,
					SealedCID:              change.Info.SealedCID,
					SealRandEpoch:          change.Info.SealRandEpoch,
					DealIDs:                change.Info.DealIDs,
					Expiration:             change.Info.Expiration,
					ReplaceCapacity:        change.Info.ReplaceCapacity,
					ReplaceSectorDeadline:  change.Info.ReplaceSectorDeadline,
					ReplaceSectorPartition: change.Info.ReplaceSectorPartition,
					ReplaceSectorNumber:    change.Info.ReplaceSectorNumber,
				},
				PreCommitDeposit:   change.PreCommitDeposit,
				PreCommitEpoch:     change.PreCommitEpoch,
				DealWeight:         change.DealWeight,
				VerifiedDealWeight: change.VerifiedDealWeight,
			},
		}
		idx++
	}
	return out, nil
}

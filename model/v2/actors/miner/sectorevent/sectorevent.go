package sectorevent

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lily/tasks/actorstate/miner/extraction"
)

// TODO: this extractor is overloaded. It should be broken up into two parts, one for Precommit events and one for Sector events
// the resulting structures should contain the full precommit/sector on-chain info structure.
// doing this will allow the sectors, precomit, and sector deals extractors to move to transformers than process these event types.

var log = logging.Logger("sectorevents")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&SectorEvent{}, Extract)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&SectorEvent{}, supportedActors)
}

var _ v2.LilyModel = (*SectorEvent)(nil)

type SectorEventType int64

const (
	CommitCapacityAdded SectorEventType = iota
	SectorAdded

	SectorExtended
	SectorSnapped

	SectorFaulted
	SectorRecovering
	SectorRecovered

	SectorExpired
	SectorTerminated
)

func (e SectorEventType) String() string {
	switch e {
	case CommitCapacityAdded:
		return "COMMIT_CAPACITY_ADDED"
	case SectorAdded:
		return "SECTOR_ADDED"
	case SectorExtended:
		return "SECTOR_EXTENDED"
	case SectorSnapped:
		return "SECTOR_SNAPPED"
	case SectorFaulted:
		return "SECTOR_FAULTED"
	case SectorRecovering:
		return "SECTOR_RECOVERING"
	case SectorRecovered:
		return "SECTOR_RECOVERED"
	case SectorExpired:
		return "SECTOR_EXPIRED"
	case SectorTerminated:
		return "SECTOR_TERMINATED"
	}
	panic(fmt.Sprintf("unhanded type %d developer error", e))
}

type SectorEvent struct {
	Height                abi.ChainEpoch
	StateRoot             cid.Cid
	Miner                 address.Address
	Event                 SectorEventType
	SectorNumber          abi.SectorNumber
	SealProof             abi.RegisteredSealProof
	SealedCID             cid.Cid
	DealIDs               []abi.DealID
	Activation            abi.ChainEpoch
	Expiration            abi.ChainEpoch
	DealWeight            abi.DealWeight
	VerifiedDealWeight    abi.DealWeight
	InitialPledge         abi.TokenAmount
	ExpectedDayReward     abi.TokenAmount
	ExpectedStoragePledge abi.TokenAmount
	ReplacedSectorAge     abi.ChainEpoch
	ReplacedDayReward     abi.TokenAmount
	SectorKeyCID          cid.Cid
}

func (t *SectorEvent) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(SectorEvent{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (t *SectorEvent) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func Extract(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	extState, err := extraction.LoadMinerStates(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	var (
		sectorChanges      = miner.MakeSectorChanges()
		sectorStateChanges = &SectorStateEvents{
			Removed:    []*miner.SectorOnChainInfo{},
			Recovering: []*miner.SectorOnChainInfo{},
			Faulted:    []*miner.SectorOnChainInfo{},
			Recovered:  []*miner.SectorOnChainInfo{},
		}
	)
	if extState.ParentState() == nil {
		// If the miner doesn't have previous state list all of its current sectors and precommits
		sectors, err := extState.CurrentState().LoadSectors(nil)
		if err != nil {
			return nil, fmt.Errorf("loading miner sectors: %w", err)
		}

		for _, sector := range sectors {
			sectorChanges.Added = append(sectorChanges.Added, *sector)
		}

	} else {
		// If the miner has previous state compute the list of new sectors and precommit in its current state.
		grp, grpCtx := errgroup.WithContext(ctx)
		grp.Go(func() error {
			start := time.Now()
			// collect changes made to miner sector array (AMT)
			sectorChanges, err = api.DiffSectors(grpCtx, a.Address, a.Current, a.Executed, extState.ParentState(), extState.CurrentState())
			if err != nil {
				return fmt.Errorf("diffing sectors %w", err)
			}
			log.Debugw("diff sectors", "miner", a.Address, "duration", time.Since(start))
			return nil
		})
		grp.Go(func() error {
			start := time.Now()
			// collect changes made to miner sectors across all miner partition states
			sectorStateChanges, err = DiffMinerSectorStates(grpCtx, extState)
			if err != nil {
				return fmt.Errorf("diffing sector states %w", err)
			}
			log.Debugw("diff sector states", "miner", a.Address, "duration", time.Since(start))
			return nil
		})
		if err := grp.Wait(); err != nil {
			return nil, err
		}
	}

	// transform the sector events to a model.
	sectorEventModel, err := ExtractSectorEvents(extState, sectorChanges, sectorStateChanges)
	if err != nil {
		return nil, err
	}

	return sectorEventModel, nil
}

// ExtractSectorEvents transforms sectorChanges, preCommitChanges, and sectorStateChanges to a MinerSectorEventList.
func ExtractSectorEvents(extState extraction.State, sectorChanges *miner.SectorChanges, sectorStateChanges *SectorStateEvents) ([]v2.LilyModel, error) {
	sectorStateEvents, err := ExtractMinerSectorStateEvents(extState, sectorStateChanges)
	if err != nil {
		return nil, err
	}

	sectorEvents := ExtractMinerSectorEvents(extState, sectorChanges)

	out := make([]v2.LilyModel, 0, len(sectorEvents)+len(sectorStateEvents))
	out = append(out, sectorEvents...)
	out = append(out, sectorStateEvents...)

	return out, nil
}

func getSectorKey(s miner.SectorOnChainInfo) cid.Cid {
	if s.SectorKeyCID != nil {
		return *s.SectorKeyCID
	}
	return cid.Undef
}

// ExtractMinerSectorStateEvents transforms the removed, recovering, faulted, and recovered sectors from `events` to a
// MinerSectorEventList.
func ExtractMinerSectorStateEvents(extState extraction.State, events *SectorStateEvents) ([]v2.LilyModel, error) {
	idx := 0
	out := make([]v2.LilyModel, len(events.Removed)+len(events.Recovering)+len(events.Faulted)+len(events.Recovered))

	for _, sector := range events.Removed {
		out[idx] = &SectorEvent{
			Height:                extState.CurrentTipSet().Height(),
			StateRoot:             extState.CurrentTipSet().ParentState(),
			Miner:                 extState.Address(),
			Event:                 SectorTerminated,
			SectorNumber:          sector.SectorNumber,
			SealProof:             sector.SealProof,
			SealedCID:             sector.SealedCID,
			DealIDs:               sector.DealIDs,
			Activation:            sector.Activation,
			Expiration:            sector.Expiration,
			DealWeight:            sector.DealWeight,
			VerifiedDealWeight:    sector.VerifiedDealWeight,
			InitialPledge:         sector.InitialPledge,
			ExpectedDayReward:     sector.ExpectedDayReward,
			ExpectedStoragePledge: sector.ExpectedStoragePledge,
			ReplacedSectorAge:     sector.ReplacedSectorAge,
			ReplacedDayReward:     sector.ReplacedDayReward,
			SectorKeyCID:          getSectorKey(*sector),
		}
		idx++
	}
	for _, sector := range events.Faulted {
		out[idx] = &SectorEvent{
			Height:                extState.CurrentTipSet().Height(),
			StateRoot:             extState.CurrentTipSet().ParentState(),
			Miner:                 extState.Address(),
			Event:                 SectorFaulted,
			SectorNumber:          sector.SectorNumber,
			SealProof:             sector.SealProof,
			SealedCID:             sector.SealedCID,
			DealIDs:               sector.DealIDs,
			Activation:            sector.Activation,
			Expiration:            sector.Expiration,
			DealWeight:            sector.DealWeight,
			VerifiedDealWeight:    sector.VerifiedDealWeight,
			InitialPledge:         sector.InitialPledge,
			ExpectedDayReward:     sector.ExpectedDayReward,
			ExpectedStoragePledge: sector.ExpectedStoragePledge,
			ReplacedSectorAge:     sector.ReplacedSectorAge,
			ReplacedDayReward:     sector.ReplacedDayReward,
			SectorKeyCID:          getSectorKey(*sector),
		}
		idx++
	}
	for _, sector := range events.Recovering {
		out[idx] = &SectorEvent{
			Height:                extState.CurrentTipSet().Height(),
			StateRoot:             extState.CurrentTipSet().ParentState(),
			Miner:                 extState.Address(),
			Event:                 SectorRecovering,
			SectorNumber:          sector.SectorNumber,
			SealProof:             sector.SealProof,
			SealedCID:             sector.SealedCID,
			DealIDs:               sector.DealIDs,
			Activation:            sector.Activation,
			Expiration:            sector.Expiration,
			DealWeight:            sector.DealWeight,
			VerifiedDealWeight:    sector.VerifiedDealWeight,
			InitialPledge:         sector.InitialPledge,
			ExpectedDayReward:     sector.ExpectedDayReward,
			ExpectedStoragePledge: sector.ExpectedStoragePledge,
			ReplacedSectorAge:     sector.ReplacedSectorAge,
			ReplacedDayReward:     sector.ReplacedDayReward,
			SectorKeyCID:          getSectorKey(*sector),
		}
		idx++
	}
	for _, sector := range events.Recovered {
		out[idx] = &SectorEvent{
			Height:                extState.CurrentTipSet().Height(),
			StateRoot:             extState.CurrentTipSet().ParentState(),
			Miner:                 extState.Address(),
			Event:                 SectorRecovered,
			SectorNumber:          sector.SectorNumber,
			SealProof:             sector.SealProof,
			SealedCID:             sector.SealedCID,
			DealIDs:               sector.DealIDs,
			Activation:            sector.Activation,
			Expiration:            sector.Expiration,
			DealWeight:            sector.DealWeight,
			VerifiedDealWeight:    sector.VerifiedDealWeight,
			InitialPledge:         sector.InitialPledge,
			ExpectedDayReward:     sector.ExpectedDayReward,
			ExpectedStoragePledge: sector.ExpectedStoragePledge,
			ReplacedSectorAge:     sector.ReplacedSectorAge,
			ReplacedDayReward:     sector.ReplacedDayReward,
			SectorKeyCID:          getSectorKey(*sector),
		}
		idx++
	}

	return out, nil
}

// ExtractMinerSectorEvents transforms the added, extended and snapped sectors from `sectors` to a MinerSectorEventList.
func ExtractMinerSectorEvents(extState extraction.State, sectors *miner.SectorChanges) []v2.LilyModel {
	out := make([]v2.LilyModel, 0, len(sectors.Added)+len(sectors.Extended)+len(sectors.Snapped))

	// track sector add and commit-capacity add
	for _, add := range sectors.Added {
		event := SectorAdded
		if len(add.DealIDs) == 0 {
			event = CommitCapacityAdded
		}
		out = append(out, &SectorEvent{
			Height:                extState.CurrentTipSet().Height(),
			StateRoot:             extState.CurrentTipSet().ParentState(),
			Miner:                 extState.Address(),
			Event:                 event,
			SectorNumber:          add.SectorNumber,
			SealProof:             add.SealProof,
			SealedCID:             add.SealedCID,
			DealIDs:               add.DealIDs,
			Activation:            add.Activation,
			Expiration:            add.Expiration,
			DealWeight:            add.DealWeight,
			VerifiedDealWeight:    add.VerifiedDealWeight,
			InitialPledge:         add.InitialPledge,
			ExpectedDayReward:     add.ExpectedDayReward,
			ExpectedStoragePledge: add.ExpectedStoragePledge,
			ReplacedSectorAge:     add.ReplacedSectorAge,
			ReplacedDayReward:     add.ReplacedDayReward,
			SectorKeyCID:          getSectorKey(add),
		})
	}

	// sector extension events
	for _, mod := range sectors.Extended {
		out = append(out, &SectorEvent{
			Height:                extState.CurrentTipSet().Height(),
			Miner:                 extState.Address(),
			StateRoot:             extState.CurrentTipSet().ParentState(),
			Event:                 SectorExtended,
			SectorNumber:          mod.To.SectorNumber,
			SealProof:             mod.To.SealProof,
			SealedCID:             mod.To.SealedCID,
			DealIDs:               mod.To.DealIDs,
			Activation:            mod.To.Activation,
			Expiration:            mod.To.Expiration,
			DealWeight:            mod.To.DealWeight,
			VerifiedDealWeight:    mod.To.VerifiedDealWeight,
			InitialPledge:         mod.To.InitialPledge,
			ExpectedDayReward:     mod.To.ExpectedDayReward,
			ExpectedStoragePledge: mod.To.ExpectedStoragePledge,
			ReplacedSectorAge:     mod.To.ReplacedSectorAge,
			ReplacedDayReward:     mod.To.ReplacedDayReward,
			SectorKeyCID:          getSectorKey(mod.To),
		})
	}

	// sector snapped events
	for _, snap := range sectors.Snapped {
		out = append(out, &SectorEvent{
			Height:                extState.CurrentTipSet().Height(),
			Miner:                 extState.Address(),
			StateRoot:             extState.CurrentTipSet().ParentState(),
			Event:                 SectorSnapped,
			SectorNumber:          snap.To.SectorNumber,
			SealProof:             snap.To.SealProof,
			SealedCID:             snap.To.SealedCID,
			DealIDs:               snap.To.DealIDs,
			Activation:            snap.To.Activation,
			Expiration:            snap.To.Expiration,
			DealWeight:            snap.To.DealWeight,
			VerifiedDealWeight:    snap.To.VerifiedDealWeight,
			InitialPledge:         snap.To.InitialPledge,
			ExpectedDayReward:     snap.To.ExpectedDayReward,
			ExpectedStoragePledge: snap.To.ExpectedStoragePledge,
			ReplacedSectorAge:     snap.To.ReplacedSectorAge,
			ReplacedDayReward:     snap.To.ReplacedDayReward,
			SectorKeyCID:          getSectorKey(snap.To),
		})
	}

	return out
}

// SectorStateEvents contains bitfields for sectors that were removed, recovered, faulted, and recovering.
type SectorStateEvents struct {
	// Removed sectors this epoch
	Removed []*miner.SectorOnChainInfo
	// Recovering sectors this epoch
	Recovering []*miner.SectorOnChainInfo
	// Faulted sectors this epoch
	Faulted []*miner.SectorOnChainInfo
	// Recovered sectors this epoch
	Recovered []*miner.SectorOnChainInfo
}

// DiffMinerSectorStates loads the SectorStates for the current and parent miner states in parallel from `extState`.
// Then compares current and parent SectorStates to produce a SectorStateEvents structure containing all sectors that are
// removed, recovering, faulted, and recovered for the state transition from parent miner state to current miner state.
func DiffMinerSectorStates(ctx context.Context, extState extraction.State) (*SectorStateEvents, error) {
	var (
		previous, current *SectorStates
		err               error
	)

	// load previous and current miner sector states in parallel
	grp := errgroup.Group{}
	grp.Go(func() error {
		previous, err = LoadSectorState(extState.ParentState())
		if err != nil {
			return fmt.Errorf("loading previous sector states %w", err)
		}
		return nil
	})
	grp.Go(func() error {
		current, err = LoadSectorState(extState.CurrentState())
		if err != nil {
			return fmt.Errorf("loading current sector states %w", err)
		}
		return nil
	})
	// if either load operation fails abort
	if err := grp.Wait(); err != nil {
		return nil, err
	}

	out := &SectorStateEvents{}
	grp = errgroup.Group{}
	grp.Go(func() error {
		// previous live sector minus current live sectors are sectors removed this epoch.
		removed, err := bitfield.SubtractBitField(previous.Live, current.Live)
		if err != nil {
			return fmt.Errorf("comparing previous live sectors to current live sectors %w", err)
		}
		sectors, err := extState.CurrentState().LoadSectors(&removed)
		if err != nil {
			return err
		}
		out.Removed = sectors
		return nil
	})

	grp.Go(func() error {
		// current recovering sectors minus previous recovering sectors are sectors recovering this epoch.
		recovering, err := bitfield.SubtractBitField(current.Recovering, previous.Recovering)
		if err != nil {
			return fmt.Errorf("comparing current recovering sectors to previous recovering sectors %w", err)
		}
		sectors, err := extState.CurrentState().LoadSectors(&recovering)
		if err != nil {
			return err
		}
		out.Recovering = sectors
		return nil
	})

	grp.Go(func() error {
		// current faulty sectors minus previous faulty sectors are sectors faulted this epoch.
		faulted, err := bitfield.SubtractBitField(current.Faulty, previous.Faulty)
		if err != nil {
			return fmt.Errorf("comparing current faulty sectors to previous faulty sectors %w", err)
		}
		sectors, err := extState.CurrentState().LoadSectors(&faulted)
		if err != nil {
			return err
		}
		out.Faulted = sectors
		return nil
	})

	grp.Go(func() error {
		// previous faulty sectors matching (intersect) active sectors are sectors recovered this epoch.
		recovered, err := bitfield.IntersectBitField(previous.Faulty, current.Active)
		if err != nil {
			return fmt.Errorf("comparing previous faulty sectors to current active sectors %w", err)
		}
		sectors, err := extState.CurrentState().LoadSectors(&recovered)
		if err != nil {
			return err
		}
		out.Faulted = sectors
		return nil
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// SectorStates contains a set of bitfields for active, live, fault, and recovering sectors.
type SectorStates struct {
	// Active sectors are those that are neither terminated nor faulty nor unproven, i.e. actively contributing power.
	Active bitfield.BitField
	// Live sectors are those that are not terminated (but may be faulty).
	Live bitfield.BitField
	// Faulty contains a subset of sectors detected/declared faulty and not yet recovered (excl. from PoSt).
	Faulty bitfield.BitField
	// Recovering contains a subset of faulty sectors expected to recover on next PoSt.
	Recovering bitfield.BitField
}

// LoadSectorState loads all sectors from a miners partitions and returns a SectorStates structure containing individual
// bitfields for all active, live, faulty and recovering sector.
func LoadSectorState(state miner.State) (*SectorStates, error) {
	sectorStates := &SectorStates{}
	// iterate the sector states
	activeC := make(chan bitfield.BitField)
	liveC := make(chan bitfield.BitField)
	faultyC := make(chan bitfield.BitField)
	recoveringC := make(chan bitfield.BitField)
	grp := errgroup.Group{}
	if err := state.ForEachDeadline(func(_ uint64, dl miner.Deadline) error {
		grp.Go(func() error {
			return dl.ForEachPartition(func(_ uint64, part miner.Partition) error {
				grp.Go(func() error {
					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					activeC <- active

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					liveC <- live

					faulty, err := part.FaultySectors()
					if err != nil {
						return err
					}
					faultyC <- faulty

					recovering, err := part.RecoveringSectors()
					if err != nil {
						return err
					}
					recoveringC <- recovering

					return nil
				})
				return nil
			})
		})
		return nil
	}); err != nil {
		return nil, err
	}
	var (
		err             error
		totalActive     []bitfield.BitField
		totalLive       []bitfield.BitField
		totalFaulty     []bitfield.BitField
		totalRecovering []bitfield.BitField
	)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case res := <-activeC:
				totalActive = append(totalActive, res)
			case res := <-liveC:
				totalLive = append(totalLive, res)
			case res := <-faultyC:
				totalFaulty = append(totalFaulty, res)
			case res := <-recoveringC:
				totalRecovering = append(totalRecovering, res)
			case <-done:
				return
			}
		}
	}()
	if err := grp.Wait(); err != nil {
		return nil, err
	}
	done <- struct{}{}
	sectorStates.Active, err = bitfield.MultiMerge(totalActive...)
	if err != nil {
		return nil, err
	}
	sectorStates.Live, err = bitfield.MultiMerge(totalLive...)
	if err != nil {
		return nil, err
	}
	sectorStates.Faulty, err = bitfield.MultiMerge(totalFaulty...)
	if err != nil {
		return nil, err
	}
	sectorStates.Recovering, err = bitfield.MultiMerge(totalRecovering...)
	if err != nil {
		return nil, err
	}

	return sectorStates, nil
}

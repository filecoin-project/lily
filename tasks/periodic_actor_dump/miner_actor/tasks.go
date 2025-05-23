package mineractordump

import (
	"context"
	"fmt"
	"sync"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	builtinminer "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actordumps"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/chain/types"
)

var log = logging.Logger("lily/tasks/mineractordump")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) updateAddressFromID(ctx context.Context, current *types.TipSet, minerDumpObj *actordumps.MinerActorDump) error {
	// Owner Address
	ownerAddr, err := address.NewFromString(minerDumpObj.OwnerID)
	if err != nil {
		return err
	}
	ownerActor, err := p.node.ActorInfo(ctx, ownerAddr, current.Key())
	if err != nil {
		return err
	}
	minerDumpObj.OwnerAddress = ownerActor.Actor.DelegatedAddress.String()

	// Worker Address
	workerAddr, err := address.NewFromString(minerDumpObj.WorkerID)
	if err != nil {
		return err
	}
	workerActor, err := p.node.ActorInfo(ctx, workerAddr, current.Key())
	if err != nil {
		return err
	}
	minerDumpObj.WorkerAddress = workerActor.Actor.DelegatedAddress.String()

	// Beneficiary Address
	beneficiaryAddr, err := address.NewFromString(minerDumpObj.Beneficiary)
	if err != nil {
		return err
	}
	beneficiaryWorkerActor, err := p.node.ActorInfo(ctx, beneficiaryAddr, current.Key())
	if err != nil {
		return err
	}
	minerDumpObj.BeneficiaryAddress = beneficiaryWorkerActor.Actor.DelegatedAddress.String()

	return nil
}

func (p *Task) ProcessPeriodicActorDump(ctx context.Context, current *types.TipSet, actors tasks.ActorStatesByType) (model.Persistable, *visormodel.ProcessingReport, error) {
	_, span := otel.Tracer("").Start(ctx, "ProcessPeriodicActorDump")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("processor", "miner_actor_state_dump"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	log.Infof("Size of Power Actors: %v", len(actors[manifest.PowerKey]))

	blockReward, err := util.GetBlockReward(ctx, p.node, current.Key(), p.node.Store())
	if err != nil {
		log.Errorf("Error at getting the block reward: %v", err)
	}

	out := make(actordumps.MinerActorDumpList, 0)
	errs := []error{}

	var (
		mu sync.Mutex
	)

	// Create a semaphore channel with a buffer size of 10 to limit concurrency.
	sem := make(chan struct{}, 10)

	// Create an errgroup with the parent context.
	g, ctx := errgroup.WithContext(ctx)

	for _, actor := range actors[manifest.PowerKey] {
		powerState, err := power.Load(p.node.Store(), actor)
		if err != nil {
			log.Errorf("Error at loading power state: [actor cid: %v] err: %v", actor.Code.String(), err)
			errs = append(errs, err)
			continue
		}

		err = powerState.ForEachClaim(func(miner address.Address, claim power.Claim) error {
			// Capture loop variables.
			minerLocal := miner
			claimLocal := claim

			// Acquire a token from the semaphore to ensure only 10 goroutines run concurrently.
			sem <- struct{}{}

			// Launch a goroutine for each claim.
			g.Go(func() error {
				// Ensure the token is released when this goroutine is done.
				defer func() { <-sem }()

				minerDumpObj := &actordumps.MinerActorDump{
					Height:          int64(current.Height()),
					StateRoot:       current.ParentState().String(),
					MinerID:         minerLocal.String(),
					RawBytePower:    claimLocal.RawBytePower.String(),
					QualityAdjPower: claimLocal.QualityAdjPower.String(),
				}

				// Update the minerInfo Field into dump model
				minerActor, err := p.node.ActorInfo(ctx, minerLocal, current.Key())
				if err != nil {
					return err
				}
				minerDumpObj.MinerAddress = minerActor.Actor.DelegatedAddress.String()

				minerState, err := builtinminer.Load(p.node.Store(), minerActor.Actor)
				if err != nil {
					return err
				}

				sectors, err := queryMinerActiveSectorsFromChain(minerState)
				if err == nil {

					// For each active sector, calculate and print the termination fee.
					if minerState.ActorVersion() >= actorstypes.Version16 {
						terminationFee := getTerminationFeeForMinerV2(current.Height(), sectors, blockReward)
						minerDumpObj.TerminationFeeV2 = terminationFee.String()
						minerDumpObj.TerminationFee = big.Zero().String()
					} else {
						terminationFee := getTerminationFeeForMiner(current.Height(), sectors)
						minerDumpObj.TerminationFee = terminationFee.String()
						minerDumpObj.TerminationFeeV2 = big.Zero().String()
					}

					// Calculate the daily fee for each sector and sum them up.
					dailyFee := sumDailyFeeForMiner(sectors)
					minerDumpObj.DailyFee = dailyFee.String()
				} else {
					log.Errorf("Error at getting the active sectors: %v", err)
				}

				// Stop the useless fields
				// err = minerDumpObj.UpdateMinerInfo(minerState)
				// if err != nil {
				// 	return err
				// }

				err = minerDumpObj.UpdateBalanceInfo(minerActor.Actor, minerState)
				if err != nil {
					log.Errorf("Error at updating balance info: %v", err)
				}

				// err = p.updateAddressFromID(ctx, current, minerDumpObj)
				// if err != nil {
				// 	log.Error("Error at getting getting the actor address by actor id: %v", err)
				// }
				mu.Lock()
				out = append(out, minerDumpObj)
				mu.Unlock()

				return nil
			})

			return nil
		})

		// Wait for all spawned goroutines to complete.
		if err := g.Wait(); err != nil {
			errs = append(errs, err)
		}

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		report.ErrorsDetected = fmt.Errorf("%v", errs)
	}

	return model.PersistableList{out}, report, nil
}

const (
	EPOCHS_IN_DAY                         = 2880
	SECTOR_EXPIRATION_DAY_THRESHOLD       = 360
	LIFETIME_CAP                          = 140
	TERMINATION_REWARD_FACTOR_DENOM       = 2
	CONTINUED_FAULT_FACTOR_NUM            = 351
	CONTINUED_FAULT_FACTOR_DENOM          = 100
	FROZEN_DURATION_AFTER_CONSENSUS_FAULT = 900
)

func getTerminationFeeForMinerV2(currentEpoch abi.ChainEpoch, sectors []*miner.SectorOnChainInfo, blockReward big.Int) abi.TokenAmount {
	// For each active sector, calculate and print the termination fee.
	var totalFee big.Int
	totalFee = big.Zero()
	for _, sector := range sectors {
		fee := calculateTerminateFeeForSectorV2(currentEpoch, sector, blockReward)

		log.Infof("Sector %d termination fee v2: %s\n", sector.SectorNumber, fee.String())
		totalFee = big.Add(totalFee, fee)
	}

	log.Infof("Total termination fee v2: %s\n", totalFee.String())
	return totalFee
}

func getTerminationFeeForMiner(currentEpoch abi.ChainEpoch, sectors []*miner.SectorOnChainInfo) abi.TokenAmount {
	// For each active sector, calculate and print the termination fee.
	var totalFee big.Int
	totalFee = big.Zero()
	for _, sector := range sectors {
		fee := calculateTerminateFeeForSector(currentEpoch, sector)

		log.Infof("Sector %d termination fee: %s\n", sector.SectorNumber, fee.String())
		totalFee = big.Add(totalFee, fee)
	}

	log.Infof("Total termination fee: %s\n", totalFee.String())
	return totalFee
}

func calculateTerminateFeeForSectorV2(currentEpoch abi.ChainEpoch, sector *miner.SectorOnChainInfo, blockReward big.Int) abi.TokenAmount {
	// sectorTerminateFee = max(a, b, c)

	// a = initialPledge * 8.5% * min(1, activatedDays / 140)
	// initialPledge * 8.5%
	simpleTermFee := big.Div(big.Mul(sector.InitialPledge, big.NewInt(85)), big.NewInt(1000))

	// activatedDays := (currentEpoch - sector.Activation) / EPOCHS_IN_DAY
	activatedDays := big.Div(big.NewInt(int64(currentEpoch-sector.Activation)), big.NewInt(int64(EPOCHS_IN_DAY)))

	// a
	// age_factor = min(1, activated_days / TERMINATION_LIFETIME_CAP)
	durationTerminationFee := big.Div(big.Mul(simpleTermFee, activatedDays), big.NewInt(140))
	if activatedDays.GreaterThan(big.NewInt(140)) {
		durationTerminationFee = simpleTermFee
	}

	// b = initialPledge * 2%
	minimumFeeAbs := big.Div(big.Mul(sector.InitialPledge, big.NewInt(2)), big.NewInt(100))

	// c = faultFee * 105%  (faultFee ~= sector's 3.5 day BR)
	// - The faultFee is essentially 3.5 days’ block rewards for the sector.
	// faultFee = blockReward * 3.5
	faultFee := big.Div(big.Mul(blockReward, big.NewInt(7)), big.NewInt(2))
	minimumFeeFF := big.Div(big.Mul(faultFee, big.NewInt(105)), big.NewInt(100))

	return big.Max(big.Max(durationTerminationFee, minimumFeeAbs), minimumFeeFF)
}

func calculateTerminateFeeForSector(currentEpoch abi.ChainEpoch, sector *miner.SectorOnChainInfo) abi.TokenAmount {
	// Set default values to 0 if any of the fields are nil
	expectedDayReward := big.Zero()
	if sector.ExpectedDayReward != nil && !sector.ExpectedDayReward.Nil() {
		expectedDayReward = *sector.ExpectedDayReward
	}

	replacedDayReward := big.Zero()
	if sector.ReplacedDayReward != nil && !sector.ReplacedDayReward.Nil() {
		replacedDayReward = *sector.ReplacedDayReward
	}

	expectedStoragePledge := big.Zero()
	if sector.ExpectedStoragePledge != nil && !sector.ExpectedStoragePledge.Nil() {
		expectedStoragePledge = *sector.ExpectedStoragePledge
	}

	// lifetime_cap in epochs.
	lifetimeCap := abi.ChainEpoch(LIFETIME_CAP * EPOCHS_IN_DAY)
	// How long has the sector been in power? Cap this value.
	cappedSectorAge := min(currentEpoch-sector.PowerBaseEpoch, lifetimeCap)
	// expected_reward = ExpectedDayReward * cappedSectorAge
	expectedReward := big.Mul(expectedDayReward, big.NewInt(int64(cappedSectorAge)))
	// relevant_replaced_age = min(PowerBaseEpoch - Activation, lifetimeCap - cappedSectorAge)
	relevantReplacedAge := big.NewInt(int64(min(sector.PowerBaseEpoch-sector.Activation, lifetimeCap-cappedSectorAge)))
	// Add the replaced sector's contribution.
	expectedReward = big.Add(expectedReward, big.Mul(replacedDayReward, relevantReplacedAge))
	// penalized_reward = expected_reward / TERMINATION_REWARD_FACTOR_DENOM
	penalizedReward := big.Div(expectedReward, big.NewInt(int64(TERMINATION_REWARD_FACTOR_DENOM)))
	// Termination fee = ExpectedStoragePledge + (penalized_reward / EPOCHS_IN_DAY)
	return big.Add(expectedStoragePledge, big.Div(penalizedReward, big.NewInt(int64(EPOCHS_IN_DAY))))
}

func queryMinerActiveSectorsFromChain(minerState miner.State) ([]*miner.SectorOnChainInfo, error) {
	// otherwise we use another strategy
	activeSectorsBitmap, err := miner.AllPartSectors(minerState, miner.Partition.ActiveSectors)
	if err != nil {
		return nil, err
	}
	activeSectors, err := minerState.LoadSectors(&activeSectorsBitmap)
	if err != nil {
		return nil, err
	}
	return activeSectors, nil
}

func sumDailyFeeForMiner(sectors []*miner.SectorOnChainInfo) abi.TokenAmount {
	// For each active sector, calculate and print the termination fee.
	var totalFee big.Int
	totalFee = big.Zero()
	for _, sector := range sectors {
		fee := sector.DailyFee
		if fee.Nil() {
			fee = big.Zero()
		}

		totalFee = big.Add(totalFee, fee)
	}

	return totalFee
}

package mineractordump

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	builtinminer "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
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

	out := make(actordumps.MinerActorDumpList, 0)
	errs := []error{}

	for _, actor := range actors[manifest.PowerKey] {
		powerState, err := power.Load(p.node.Store(), actor)
		if err != nil {
			log.Errorf("Error at loading power state: [actor cid: %v] err: %v", actor.Code.String(), err)
			errs = append(errs, err)
			continue
		}

		err = powerState.ForEachClaim(func(miner address.Address, claim power.Claim) error {
			minerDumpObj := &actordumps.MinerActorDump{
				Height:          int64(current.Height()),
				StateRoot:       current.ParentState().String(),
				MinerID:         miner.String(),
				RawBytePower:    claim.RawBytePower.String(),
				QualityAdjPower: claim.QualityAdjPower.String(),
			}

			// Update the minerInfo Field into dump model
			minerActor, err := p.node.ActorInfo(ctx, miner, current.Key())
			if err != nil {
				return err
			}
			minerDumpObj.MinerAddress = minerActor.Actor.DelegatedAddress.String()

			minerState, err := builtinminer.Load(p.node.Store(), minerActor.Actor)
			if err != nil {
				return err
			}

			fee := getTerminationFeeForMiner(current.Height(), minerState)
			minerDumpObj.TerminationFee = fee.String()

			err = minerDumpObj.UpdateMinerInfo(minerState)
			if err != nil {
				return err
			}

			err = minerDumpObj.UpdateBalanceInfo(minerActor.Actor, minerState)
			if err != nil {
				return err
			}

			err = p.updateAddressFromID(ctx, current, minerDumpObj)
			if err != nil {
				log.Error("Error at getting getting the actor address by actor id: %v", err)
			}
			out = append(out, minerDumpObj)

			return nil
		})

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

func getTerminationFeeForMiner(currentEpoch abi.ChainEpoch, minerState builtinminer.State) abi.TokenAmount {
	sectors, err := queryMinerActiveSectorsFromChain(minerState)
	if err != nil {
		log.Errorf("Error at querying miner active sectors: %v", err)
		return big.Zero()
	}

	// For each active sector, calculate and print the termination fee.
	var totalFee big.Int
	totalFee = big.Zero()
	for _, sector := range sectors {
		fee := calculateTerminateFeeForSector(currentEpoch, sector)

		fmt.Printf("Sector %d termination fee: %s\n", sector.SectorNumber, fee.String())
		totalFee = big.Add(totalFee, fee)
	}

	fmt.Printf("Total termination fee: %s\n", totalFee.String())
	return totalFee
}

func calculateTerminateFeeForSector(currentEpoch abi.ChainEpoch, sector *miner.SectorOnChainInfo) abi.TokenAmount {
	// lifetime_cap in epochs.
	lifetimeCap := abi.ChainEpoch(LIFETIME_CAP * EPOCHS_IN_DAY)
	// How long has the sector been in power? Cap this value.
	cappedSectorAge := min(currentEpoch-sector.PowerBaseEpoch, lifetimeCap)
	// expected_reward = ExpectedDayReward * cappedSectorAge
	expectedReward := big.Mul(sector.ExpectedDayReward, big.NewInt(int64(cappedSectorAge)))
	// relevant_replaced_age = min(PowerBaseEpoch - Activation, lifetimeCap - cappedSectorAge)
	relevantReplacedAge := big.NewInt(int64(min(sector.PowerBaseEpoch-sector.Activation, lifetimeCap-cappedSectorAge)))
	// Add the replaced sector's contribution.
	expectedReward = big.Add(expectedReward, big.Mul(sector.ReplacedDayReward, relevantReplacedAge))
	// penalized_reward = expected_reward / TERMINATION_REWARD_FACTOR_DENOM
	penalizedReward := big.Div(expectedReward, big.NewInt(int64(TERMINATION_REWARD_FACTOR_DENOM)))
	// Termination fee = ExpectedStoragePledge + (penalized_reward / EPOCHS_IN_DAY)
	return big.Add(sector.ExpectedStoragePledge, big.Div(penalizedReward, big.NewInt(int64(EPOCHS_IN_DAY))))
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

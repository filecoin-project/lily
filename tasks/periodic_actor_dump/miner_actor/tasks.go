package mineractordump

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lotus/chain/types"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	builtinminer "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actordumps"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
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
				return nil
			}
			minerDumpObj.MinerAddress = minerActor.Actor.Address.String()

			minerState, err := builtinminer.Load(p.node.Store(), minerActor.Actor)
			if err != nil {
				return nil
			}

			err = minerDumpObj.UpdateMinerInfo(minerState)
			if err != nil {
				return nil
			}

			err = minerDumpObj.UpdateBalanceRelated(minerActor.Actor, minerState)
			if err != nil {
				return nil
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

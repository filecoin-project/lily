package fevmactorstatedump

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lotus/chain/actors/builtin/evm"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/statedumps"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/tasks/fevmactorstatedump")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessPeriodicStateDump(ctx context.Context, current *types.TipSet, actors tasks.ActorStatesByType) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessPeriodicStateDump")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("processor", "fevm_actor_state_dump"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	log.Infof("Size of EVM Actors: %v", len(actors[manifest.EvmKey]))

	out := make(statedumps.FEVMActorStateDumpList, 0)
	errs := []error{}
	for _, actor := range actors[manifest.EvmKey] {
		if actor.Address == nil {
			continue
		}

		evmState, err := evm.Load(p.node.Store(), actor)
		if err != nil {
			log.Errorf("Error at loading evm state: [actor cid: %v] err: %v", actor.Code.String(), err)
			errs = append(errs, err)
			continue
		}

		ethAddress, err := ethtypes.EthAddressFromFilecoinAddress(*actor.Address)
		if err != nil {
			log.Errorf("Error at getting eth address: [actor cid: %v] err: %v", actor.Code.String(), err)
			errs = append(errs, err)
			continue
		}

		byteCode, err := evmState.GetBytecode()
		if err != nil {
			log.Errorf("Error at getting byte code: [actor cid: %v] err: %v", actor.Code.String(), err)
			errs = append(errs, err)
			continue
		}

		byteCodeHash, err := evmState.GetBytecodeHash()
		if err != nil {
			log.Errorf("Error at getting byte code hash: [actor cid: %v] err: %v", actor.Code.String(), err)
			errs = append(errs, err)
			continue
		}

		out = append(out, &statedumps.FEVMAcotrStateDump{
			Height:       int64(current.Height()),
			ActorID:      actor.Address.String(),
			EthAddress:   ethAddress.String(),
			ByteCode:     hex.EncodeToString(byteCode),
			ByteCodeHash: hex.EncodeToString(byteCodeHash[:]),
			Balance:      actor.Balance.String(),
			Nonce:        actor.Nonce,
		})

	}

	if len(errs) > 0 {
		report.ErrorsDetected = fmt.Errorf("%v", errs)
	}

	return model.PersistableList{out}, report, nil
}

package fevmcontract

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/filecoin-project/lotus/chain/actors/builtin/evm"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/fevm"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lily/lens/util"
)

var log = logging.Logger("lily/tasks/fevmcontract")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "fevm_contract"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	actorChanges, err := p.node.ActorStateChanges(ctx, current, executed)
	if err != nil {
		return nil, report, err
	}

	out := make(fevm.FEVMContractList, 0)
	errs := []error{}
	for _, change := range actorChanges {
		actor := change.Actor
		if actor.Address == nil {
			continue
		}

		if !util.IsEVMAddress(ctx, p.node, *actor.Address, current.Key()) {
			continue
		}

		evmState, err := evm.Load(p.node.Store(), &actor)
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

		out = append(out, &fevm.FEVMContract{
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
		err = fmt.Errorf("%v", errs)
	} else {
		err = nil
	}

	return model.PersistableList{out}, report, err
}

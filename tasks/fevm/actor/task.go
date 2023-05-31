package fevmactor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lily/model/fevm"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"

	builtintypes "github.com/filecoin-project/go-state-types/builtin"
)

var log = logging.Logger("lily/tasks/fevmactor")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.Key().String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("processor", "fevmactor"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	messages, err := p.node.ChainGetMessagesInTipset(ctx, ts.Key())
	if err != nil {
		log.Errorf("Error at getting messages. ts: %v, height: %v, err: %v", ts.String(), ts.Height(), err)
		return nil, report, err
	}
	errs := []error{}
	out := make(fevm.FEVMActorList, 0)
	storedCache := map[string]bool{}

	for _, message := range messages {
		if message.Message == nil {
			continue
		}

		// Prevent from duplicating
		_, stored := storedCache[message.Message.From.String()]
		if stored {
			continue
		}

		// Only handle the evm actor creation message
		if message.Message.To != builtintypes.EthereumAddressManagerActorAddr {
			continue
		}

		if util.IsEVMAddress(ctx, p.node, message.Message.From, ts.Key()) {
			continue
		}

		ethAddress, err := ethtypes.EthAddressFromFilecoinAddress(message.Message.From)
		if err != nil {
			log.Errorf("Error at getting eth address [address: %v] err: %v", message.Message.From.String(), err)
			continue
		}

		actor, err := p.node.Actor(ctx, message.Message.From, ts.Key())
		if err != nil {
			log.Errorf("Error at getting actor [address: %v] err: %v", message.Message.From.String(), err)
			continue
		}

		stateStr := ""
		actorState, err := p.node.ActorState(ctx, message.Message.From, ts)
		if err == nil {
			state, err := json.Marshal(actorState.State)
			if err == nil {
				stateStr = string(state)
			}
		}

		actorObj := &fevm.FEVMActor{
			Height:     int64(ts.Height()),
			ID:         actor.Address.String(),
			Code:       builtin.ActorNameByCode(actor.Code),
			StateRoot:  ts.ParentState().String(),
			Head:       actor.Head.String(),
			EthAddress: ethAddress.String(),
			State:      stateStr,
			CodeCID:    actor.Code.String(),
		}

		out = append(out, actorObj)

		// Prevent from duplicating
		storedCache[message.Message.From.String()] = true
	}

	if len(errs) > 0 {
		err = fmt.Errorf("%v", errs)
	} else {
		err = nil
	}

	return model.PersistableList{out}, report, err
}

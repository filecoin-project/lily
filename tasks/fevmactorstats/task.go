package fevmactorstats

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lily/model/fevm"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	evm2 "github.com/filecoin-project/lotus/chain/actors/builtin/evm"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/tasks/fevmactorstats")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	_, span := otel.Tracer("").Start(ctx, "ProcessTipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.Key().String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("processor", "fevmactorstats"),
		)
	}
	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
	}

	st, err := state.LoadStateTree(p.node.Store(), ts.ParentState())
	if err != nil {
		return nil, report, err
	}

	log.Infow("iterating over all actors")
	count := 0
	evmBalance := abi.NewTokenAmount(0)
	ethAccountBalance := abi.NewTokenAmount(0)
	placeholderBalance := abi.NewTokenAmount(0)
	evmCount := 0
	ethAccountCount := 0
	placeholderCount := 0
	bytecodeCIDs := []cid.Cid{}

	_ = st.ForEach(func(addr address.Address, act *types.Actor) error {
		count++

		if builtin.IsEvmActor(act.Code) {
			evmBalance = types.BigAdd(evmBalance, act.Balance)
			evmCount++
			es, err := evm2.Load(p.node.Store(), act)
			if err != nil {
				log.Errorw("fail to load evm actorcount: ", "error", err)
				return err
			}
			bcid, err := es.GetBytecodeCID()
			if err != nil {
				log.Errorw("fail to get evm bytecode: ", "error", err)
				return err
			}
			bytecodeCIDs = append(bytecodeCIDs, bcid)
		}

		if builtin.IsEthAccountActor(act.Code) {
			ethAccountBalance = types.BigAdd(ethAccountBalance, act.Balance)
			ethAccountCount++
		}

		if builtin.IsPlaceholderActor(act.Code) {
			placeholderBalance = types.BigAdd(placeholderBalance, act.Balance)
			placeholderCount++
		}

		return nil
	})

	uniqueBytecode := unique(bytecodeCIDs)

	return &fevm.FEVMActorStats{
		Height:              int64(ts.Height()),
		ContractBalance:     evmBalance.String(),
		EthAccountBalance:   ethAccountBalance.String(),
		PlaceholderBalance:  placeholderBalance.String(),
		ContractCount:       uint64(evmCount),
		UniqueContractCount: uint64(len(uniqueBytecode)),
		EthAccountCount:     uint64(ethAccountCount),
		PlaceholderCount:    uint64(placeholderCount),
	}, report, nil
}
func unique(intSlice []cid.Cid) []cid.Cid {
	keys := make(map[cid.Cid]bool)
	list := []cid.Cid{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

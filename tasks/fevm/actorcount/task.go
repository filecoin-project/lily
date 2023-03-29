package actorcount

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	evm2 "github.com/filecoin-project/lotus/chain/actors/builtin/evm"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	_, span := otel.Tracer("").Start(ctx, "ProcessTipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.Key().String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("processor", "chaineconomics"),
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
	EvmCount := 0
	EthAccountCount := 0
	PlaceholderCount := 0
	bytecodeCIDs := []cid.Cid{}

	err = st.ForEach(func(addr address.Address, act *types.Actor) error {
		if count%200000 == 0 {
			log.Infow("processed /n", count)
		}
		count++

		if builtin.IsEvmActor(act.Code) {
			EvmCount++
			e, err := evm2.Load(p.node.Store(), act)
			if err != nil {
				log.Errorw("fail to load evm actorcount: %w", err)
				return nil
			}
			bcid, err := e.GetBytecodeCID()
			bytecodeCIDs = append(bytecodeCIDs, bcid)
		}

		if builtin.IsEthAccountActor(act.Code) {
			EthAccountCount++
		}

		if builtin.IsPlaceholderActor(act.Code) {
			PlaceholderCount++
		}

		return nil
	})

	uniqueBytecodeCIDs := unique(bytecodeCIDs)
	log.Infow("# of EVM contracts: ", EvmCount)
	log.Infow("# of unqiue EVM contracts: ", len(uniqueBytecodeCIDs))
	log.Infow("b# of Eth accounts: ", EthAccountCount)
	log.Infow("# of placeholder: ", PlaceholderCount)

	return nil, report, nil
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

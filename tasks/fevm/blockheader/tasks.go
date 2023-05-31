package fevmblockheader

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/fevm"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
)

var log = logging.Logger("lily/tasks/fevmblockheader")

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
			attribute.String("processor", "fevm_block_header"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	cid, err := executed.Key().Cid()
	if err != nil {
		log.Errorf("Error at getting cid: [%v] err: %v", cid, err)
		return nil, report, err
	}

	hash, err := ethtypes.EthHashFromCid(cid)
	if err != nil {
		log.Errorf("Error at finding hash: [%v] err: %v", hash, err)
		return nil, report, err
	}

	ethBlock, err := p.node.EthGetBlockByHash(ctx, hash, false)
	if err != nil {
		log.Errorf("EthGetBlockByHash: [hash: %v] err: %v", hash.String(), err)
		return nil, report, err
	}

	if ethBlock.Number == 0 {
		log.Warn("block number == 0")
		return nil, report, err
	}
	return &fevm.FEVMBlockHeader{
		Height:           int64(executed.Height()),
		Hash:             hash.String(),
		ParentHash:       ethBlock.ParentHash.String(),
		Miner:            ethBlock.Miner.String(),
		StateRoot:        ethBlock.StateRoot.String(),
		TransactionsRoot: ethBlock.TransactionsRoot.String(),
		ReceiptsRoot:     ethBlock.ReceiptsRoot.String(),
		Difficulty:       uint64(ethBlock.Difficulty),
		Number:           uint64(ethBlock.Number),
		GasLimit:         uint64(ethBlock.GasLimit),
		GasUsed:          uint64(ethBlock.GasUsed),
		Timestamp:        uint64(ethBlock.Timestamp),
		ExtraData:        string(ethBlock.Extradata),
		MixHash:          ethBlock.MixHash.String(),
		Nonce:            ethBlock.Nonce.String(),
		BaseFeePerGas:    ethBlock.BaseFeePerGas.String(),
		Size:             uint64(ethBlock.Size),
		Sha3Uncles:       ethBlock.Sha3Uncles.String(),
	}, report, nil
}

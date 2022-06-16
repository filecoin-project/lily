package gaseconomy

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/messages"
)

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "gas_economy"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	tsMsgs, err := t.node.ExecutedAndBlockMessages(ctx, current, executed)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting executed and block messages: %w", err)
		return nil, report, nil
	}
	emsgs := tsMsgs.Executed

	var (
		exeMsgSeen        = make(map[cid.Cid]bool, len(emsgs))
		totalGasLimit     int64
		totalUniqGasLimit int64
		errorsDetected    = make([]*messages.MessageError, 0, len(emsgs))
	)

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		// calculate total gas limit of executed messages regardless of duplicates.
		for range m.Blocks {
			totalGasLimit += m.Message.GasLimit
		}

		if exeMsgSeen[m.Cid] {
			continue
		}
		exeMsgSeen[m.Cid] = true
		totalUniqGasLimit += m.Message.GasLimit
	}

	newBaseFee := store.ComputeNextBaseFee(executed.Blocks()[0].ParentBaseFee, totalUniqGasLimit, len(executed.Blocks()), executed.Height())
	baseFeeRat := new(big.Rat).SetFrac(newBaseFee.Int, new(big.Int).SetUint64(build.FilecoinPrecision))
	baseFee, _ := baseFeeRat.Float64()

	baseFeeChange := new(big.Rat).SetFrac(newBaseFee.Int, executed.Blocks()[0].ParentBaseFee.Int)
	baseFeeChangeF, _ := baseFeeChange.Float64()

	messageGasEconomyResult := &messagemodel.MessageGasEconomy{
		Height:              int64(executed.Height()),
		StateRoot:           executed.ParentState().String(),
		GasLimitTotal:       totalGasLimit,
		GasLimitUniqueTotal: totalUniqGasLimit,
		BaseFee:             baseFee,
		BaseFeeChangeLog:    math.Log(baseFeeChangeF) / math.Log(1.125),
		GasFillRatio:        float64(totalGasLimit) / float64(len(executed.Blocks())*build.BlockGasTarget),
		GasCapacityRatio:    float64(totalUniqGasLimit) / float64(len(executed.Blocks())*build.BlockGasTarget),
		GasWasteRatio:       float64(totalGasLimit-totalUniqGasLimit) / float64(len(executed.Blocks())*build.BlockGasTarget),
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return messageGasEconomyResult, report, nil
}

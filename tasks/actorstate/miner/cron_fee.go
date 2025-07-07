package miner

import (
	"bytes"
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	miner16 "github.com/filecoin-project/go-state-types/builtin/v16/miner"
	builtinminer "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	minertypes "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
)

var cronFeeLogger = logging.Logger("lily/tasks/cronfee")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{node: node}
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "vm_messages"),
		)
	}
	defer span.End()

	burnAddr, err := address.NewFromString("f099")
	if err != nil {
		return nil, nil, fmt.Errorf("parsing burn address: %w", err)
	}

	var cronParams *miner16.DeferredCronEventParams

	type minerBurn struct {
		addr    address.Address
		burn    big.Int
		fee     big.Int
		penalty big.Int
	}

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	// Calculate the expected penalty for a given power amount. This is unfortunately complicated
	// by the need to fetch total network reward and power for the current tipset.
	faultFeeForPower := func(qaPower abi.StoragePower) (abi.TokenAmount, error) {
		currentNetworkVersion := util.DefaultNetwork.Version(ctx, current.Height())
		if err != nil {
			return big.Zero(), err
		}

		return minertypes.PledgePenaltyForContinuedFault(
			currentNetworkVersion,
			lbuiltin.FilterEstimate{
				PositionEstimate: cronParams.RewardSmoothed.PositionEstimate,
				VelocityEstimate: cronParams.RewardSmoothed.VelocityEstimate,
			},
			lbuiltin.FilterEstimate{
				PositionEstimate: cronParams.QualityAdjPowerSmoothed.PositionEstimate,
				VelocityEstimate: cronParams.QualityAdjPowerSmoothed.VelocityEstimate,
			},
			qaPower,
		)
	}

	inspectMiner := func(minerAddr address.Address) (big.Int, big.Int, error) {
		minerActor, err := t.node.ActorInfo(ctx, minerAddr, current.Key())
		if err != nil {
			return big.Zero(), big.Zero(), fmt.Errorf("getting miner actor: %w", err)
		}
		minerState, err := builtinminer.Load(t.node.Store(), minerActor.Actor)
		if err != nil {
			return big.Zero(), big.Zero(), fmt.Errorf("loading miner state: %w", err)
		}

		dinfo, err := minerState.DeadlineInfo(current.Height())
		if err != nil {
			return big.Zero(), big.Zero(), fmt.Errorf("getting deadline info: %w", err)
		}

		deadline, err := minerState.LoadDeadline(dinfo.Index)
		if err != nil {
			return big.Zero(), big.Zero(), fmt.Errorf("loading deadline: %w", err)
		}

		faultFee := big.Zero()
		faultFeeQA, err := deadline.FaultyPowerQA()
		if err != nil {
			faultFeeQA = big.Zero()
			log.Errorf("getting faulty power QA: %w", err)
		}
		if !faultFeeQA.IsZero() {
			faultFee, err = faultFeeForPower(faultFeeQA)
			if err != nil {
				return big.Zero(), big.Zero(), xerrors.Errorf("getting fault fee: %w", err)
			}
		}

		expectedFee, err := deadline.DailyFee()
		if err != nil {
			expectedFee = big.Zero()
			log.Errorf("getting expected daily fee: %w", err)
		}
		// Check if the fees should be capped to 50% of expected daily reward; this is an unlikely
		// case and we could ignore it and still be correct almost all of the time.
		livePowerQA, err := deadline.LivePowerQA()
		if err != nil {
			livePowerQA = big.Zero()
			log.Errorf("getting live power QA: %w", err)
		}
		rew := miner16.ExpectedRewardForPower(cronParams.RewardSmoothed, cronParams.QualityAdjPowerSmoothed, livePowerQA, builtin.EpochsInDay)
		feeCap := big.Div(rew, big.NewInt(miner16.DailyFeeBlockRewardCapDenom))
		if feeCap.LessThan(expectedFee) {
			expectedFee = feeCap
		}

		return expectedFee, faultFee, nil
	}

	compute, err := t.node.StateCompute(ctx, current.Height(), nil, current.Key())
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("computing state at tipset: %w", err)
		return nil, report, nil
	}

	cronMinerCallsCache := make(map[string]struct{})

	burns := []*minermodel.MinerCronFee{}

	var traceBurns func(depth int, trace types.ExecutionTrace, thisExecCronMiner *minermodel.MinerCronFee) error
	traceBurns = func(depth int, trace types.ExecutionTrace, thisExecCronMiner *minermodel.MinerCronFee) error {
		if trace.Msg.From == power.Address && trace.Msg.Method == 12 {
			// cron call to miner
			if thisExecCronMiner != nil {
				if _, ok := cronMinerCallsCache[thisExecCronMiner.Address]; ok {
					return fmt.Errorf("multiple cron calls to same miner in one message: %s", thisExecCronMiner.Address)
				}
			}

			var p miner16.DeferredCronEventParams
			if err := p.UnmarshalCBOR(bytes.NewReader(trace.Msg.Params)); err != nil {
				return fmt.Errorf("unmarshalling cron params: %w", err)
			}
			cronParams = &p

			fee, penalty, err := inspectMiner(trace.Msg.To)
			if err != nil {
				return xerrors.Errorf("inspecting miner: %w", err)
			}
			thisExecCronMiner = &minermodel.MinerCronFee{
				Height:  int64(current.Height()),
				Address: trace.Msg.To.String(),
				Burn:    big.Zero().String(),
				Fee:     fee.String(),
				Penalty: penalty.String(),
			}
			burns = append(burns, thisExecCronMiner)
			cronMinerCallsCache[trace.Msg.To.String()] = struct{}{}
		} else if thisExecCronMiner != nil && trace.Msg.From.String() == thisExecCronMiner.Address && trace.Msg.To == burnAddr {
			// TODO: handle multiple burn? Shouldn't happen but maybe it should be checked?
			thisExecCronMiner.Burn = trace.Msg.Value.String()
		}

		for _, st := range trace.Subcalls {
			if err := traceBurns(depth+1, st, thisExecCronMiner); err != nil {
				return err
			}
		}

		return nil
	}

	errors := make([]error, 0)
	for _, invoc := range compute.Trace {
		trace := invoc.ExecutionTrace
		if err := traceBurns(0, trace, nil); err != nil {
			log.Errorf("error processing execution trace: %v", err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		report.ErrorsDetected = fmt.Errorf("errors processing cron fee burns: %v", errors)
	}

	result := make(minermodel.MinerCronFeeList, 0, len(burns))
	for _, burn := range burns {
		result = append(result, burn)
	}

	return result, report, nil
}

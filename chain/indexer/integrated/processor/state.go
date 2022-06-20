package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	saminer1 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	saminer2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	saminer3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	saminer4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	saminer5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	saminer6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	saminer7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"

	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	inittask "github.com/filecoin-project/lily/tasks/actorstate/init_"
	markettask "github.com/filecoin-project/lily/tasks/actorstate/market"
	minertask "github.com/filecoin-project/lily/tasks/actorstate/miner"
	multisigtask "github.com/filecoin-project/lily/tasks/actorstate/multisig"
	powertask "github.com/filecoin-project/lily/tasks/actorstate/power"
	rawtask "github.com/filecoin-project/lily/tasks/actorstate/raw"
	rewardtask "github.com/filecoin-project/lily/tasks/actorstate/reward"
	verifregtask "github.com/filecoin-project/lily/tasks/actorstate/verifreg"
	"github.com/filecoin-project/lily/tasks/blocks/drand"
	"github.com/filecoin-project/lily/tasks/blocks/headers"
	"github.com/filecoin-project/lily/tasks/blocks/parents"
	"github.com/filecoin-project/lily/tasks/chaineconomics"
	"github.com/filecoin-project/lily/tasks/consensus"
	"github.com/filecoin-project/lily/tasks/indexer"
	"github.com/filecoin-project/lily/tasks/messageexecutions/internalmessage"
	"github.com/filecoin-project/lily/tasks/messageexecutions/internalparsedmessage"
	"github.com/filecoin-project/lily/tasks/messages/blockmessage"
	"github.com/filecoin-project/lily/tasks/messages/gaseconomy"
	"github.com/filecoin-project/lily/tasks/messages/gasoutput"
	"github.com/filecoin-project/lily/tasks/messages/message"
	"github.com/filecoin-project/lily/tasks/messages/parsedmessage"
	"github.com/filecoin-project/lily/tasks/messages/receipt"
	"github.com/filecoin-project/lily/tasks/msapprovals"
)

type TipSetProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessTipSet(ctx context.Context, current *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type TipSetsProcessor interface {
	// ProcessTipSets processes sequential tipsts (a parent and a child, or an executed and a current). If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type ActorProcessor interface {
	// ProcessActors processes a set of actors. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessActors(ctx context.Context, current *types.TipSet, executed *types.TipSet, actors tasks.ActorStateChangeDiff) (model.Persistable, *visormodel.ProcessingReport, error)
}

type ReportProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessTipSet(ctx context.Context, current *types.TipSet) (visormodel.ProcessingReportList, error)
}

var log = logging.Logger("lily/index/processor")

const BuiltinTaskName = "builtin"

func New(api tasks.DataSource, name string, taskNames []string) (*StateProcessor, error) {
	taskNames = append(taskNames, BuiltinTaskName)

	processors, err := MakeProcessors(api, taskNames)
	if err != nil {
		return nil, err
	}
	return &StateProcessor{
		builtinProcessors: processors.ReportProcessors,
		tipsetProcessors:  processors.TipsetProcessors,
		tipsetsProcessors: processors.TipsetsProcessors,
		actorProcessors:   processors.ActorProcessors,
		api:               api,
		name:              name,
	}, nil
}

type StateProcessor struct {
	builtinProcessors map[string]ReportProcessor
	tipsetProcessors  map[string]TipSetProcessor
	tipsetsProcessors map[string]TipSetsProcessor
	actorProcessors   map[string]ActorProcessor

	// api used by tasks
	api tasks.DataSource

	//pwg is a wait group used internally to signal processors completion
	pwg sync.WaitGroup

	// name of the processor
	name string
}

// A Result is either some data to persist or an error which indicates that the task did not complete. Partial
// completions are possibly provided the Data contains a persistable log of the results.
type Result struct {
	Task        string
	Error       error
	Report      visormodel.ProcessingReportList
	Data        model.Persistable
	StartedAt   time.Time
	CompletedAt time.Time
}

// State executes its configured processors in parallel, processing the state in `current` and `executed. The return channel
// emits results of the state extraction closing when processing is completed. It is the responsibility of the processors
// to abort if its context is canceled.
// A list of all tasks executing is returned.
func (sp *StateProcessor) State(ctx context.Context, current, executed *types.TipSet) (chan *Result, []string) {
	ctx, span := otel.Tracer("").Start(ctx, "StateProcessor.State")

	num := len(sp.tipsetProcessors) + len(sp.actorProcessors) + len(sp.tipsetsProcessors) + len(sp.builtinProcessors)
	results := make(chan *Result, num)
	taskNames := make([]string, 0, num)

	taskNames = append(taskNames, sp.startReport(ctx, current, results)...)
	taskNames = append(taskNames, sp.startTipSet(ctx, current, results)...)
	taskNames = append(taskNames, sp.startTipSets(ctx, current, executed, results)...)
	taskNames = append(taskNames, sp.startActor(ctx, current, executed, results)...)

	go func() {
		sp.pwg.Wait()
		defer span.End()
		close(results)
	}()
	return results, taskNames
}

// startReport starts all ReportProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startReport(ctx context.Context, current *types.TipSet, results chan *Result) []string {
	start := time.Now()
	var taskNames []string
	for taskName, proc := range sp.builtinProcessors {
		name := taskName
		p := proc
		taskNames = append(taskNames, name)

		sp.pwg.Add(1)
		go func() {
			ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
			stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
			stop := metrics.Timer(ctx, metrics.ProcessingDuration)
			defer stop()

			pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
			pl.Infow("processor started")
			defer func() {
				pl.Infow("processor ended", "duration", time.Since(start))
				sp.pwg.Done()
			}()

			report, err := p.ProcessTipSet(ctx, current)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				pl.Errorw("processor error", "error", err)
				return
			}
			results <- &Result{
				Task:        name,
				Report:      report,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
	return taskNames
}

// startTipSet starts all TipSetProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startTipSet(ctx context.Context, current *types.TipSet, results chan *Result) []string {
	start := time.Now()
	var taskNames []string
	for taskName, proc := range sp.tipsetProcessors {
		name := taskName
		p := proc
		taskNames = append(taskNames, name)

		sp.pwg.Add(1)
		go func() {
			ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
			stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
			stop := metrics.Timer(ctx, metrics.ProcessingDuration)
			defer stop()

			pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
			pl.Infow("processor started")
			defer func() {
				pl.Infow("processor ended", "duration", time.Since(start))
				sp.pwg.Done()
			}()

			data, report, err := p.ProcessTipSet(ctx, current)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				pl.Errorw("processor error", "error", err)
				return
			}
			results <- &Result{
				Task:        name,
				Report:      visormodel.ProcessingReportList{report},
				Data:        data,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
	return taskNames
}

// startTipSets starts all TipSetsProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startTipSets(ctx context.Context, current, executed *types.TipSet, results chan *Result) []string {
	start := time.Now()
	var taskNames []string
	for taskName, proc := range sp.tipsetsProcessors {
		name := taskName
		p := proc
		taskNames = append(taskNames, name)

		sp.pwg.Add(1)
		go func() {
			ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
			stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
			stop := metrics.Timer(ctx, metrics.ProcessingDuration)
			defer stop()

			pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
			pl.Infow("processor started")
			defer func() {
				pl.Infow("processor ended", "duration", time.Since(start))
				sp.pwg.Done()
			}()

			data, report, err := p.ProcessTipSets(ctx, current, executed)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				pl.Errorw("processor error", "error", err)
				return
			}
			results <- &Result{
				Task:        name,
				Report:      visormodel.ProcessingReportList{report},
				Data:        data,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
	return taskNames
}

// startActor starts all ActorProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startActor(ctx context.Context, current, executed *types.TipSet, results chan *Result) []string {

	if len(sp.actorProcessors) == 0 {
		return nil
	}

	var taskNames []string
	for name := range sp.actorProcessors {
		taskNames = append(taskNames, name)
	}

	sp.pwg.Add(len(sp.actorProcessors))
	go func() {
		start := time.Now()
		changes, err := sp.api.ActorStateChanges(ctx, current, executed)
		if err != nil {
			// report all processor tasks as failed
			for name := range sp.actorProcessors {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:  name,
					Error: nil,
					Report: visormodel.ProcessingReportList{&visormodel.ProcessingReport{
						Height:         int64(current.Height()),
						StateRoot:      current.ParentState().String(),
						Reporter:       sp.name,
						Task:           name,
						StartedAt:      start,
						CompletedAt:    time.Now(),
						Status:         visormodel.ProcessingStatusError,
						ErrorsDetected: err,
					}},
					Data:        nil,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				sp.pwg.Done()
			}
			return
		}

		for taskName, proc := range sp.actorProcessors {
			name := taskName
			p := proc

			go func() {
				ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
				stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
				stop := metrics.Timer(ctx, metrics.ProcessingDuration)
				defer stop()

				pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
				pl.Infow("processor started")
				defer func() {
					pl.Infow("processor ended", "duration", time.Since(start))
					sp.pwg.Done()
				}()

				data, report, err := p.ProcessActors(ctx, current, executed, changes)
				if err != nil {
					stats.Record(ctx, metrics.ProcessingFailure.M(1))
					results <- &Result{
						Task:        name,
						Error:       err,
						StartedAt:   start,
						CompletedAt: time.Now(),
					}
					pl.Warnw("processor error", "error", err)
					return
				}
				results <- &Result{
					Task:        name,
					Report:      visormodel.ProcessingReportList{report},
					Data:        data,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
			}()
		}
	}()
	return taskNames
}

type IndexerProcessors struct {
	TipsetProcessors  map[string]TipSetProcessor
	TipsetsProcessors map[string]TipSetsProcessor
	ActorProcessors   map[string]ActorProcessor
	ReportProcessors  map[string]ReportProcessor
}

func MakeProcessors(api tasks.DataSource, indexerTasks []string) (*IndexerProcessors, error) {
	out := &IndexerProcessors{
		TipsetProcessors:  make(map[string]TipSetProcessor),
		TipsetsProcessors: make(map[string]TipSetsProcessor),
		ActorProcessors:   make(map[string]ActorProcessor),
		ReportProcessors:  make(map[string]ReportProcessor),
	}
	for _, t := range indexerTasks {
		switch t {
		//
		// miners
		//
		case tasktype.MinerCurrentDeadlineInfo:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.DeadlineInfoExtractor{},
			))
		case tasktype.MinerFeeDebt:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.FeeDebtExtractor{},
			))
		case tasktype.MinerInfo:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.InfoExtractor{},
			))
		case tasktype.MinerLockedFund:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.LockedFundsExtractor{},
			))
		case tasktype.MinerPreCommitInfo:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.PreCommitInfoExtractor{},
			))
		case tasktype.MinerSectorDeal:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.SectorDealsExtractor{},
			))
		case tasktype.MinerSectorEvent:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.SectorEventsExtractor{},
			))
		case tasktype.MinerSectorPost:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.PoStExtractor{},
			))
		case tasktype.MinerSectorInfoV1_6:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewCustomTypedActorExtractorMap(
				map[cid.Cid][]actorstate.ActorStateExtractor{
					saminer1.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
					saminer2.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
					saminer3.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
					saminer4.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
					saminer5.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
					saminer6.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
				},
			))
		case tasktype.MinerSectorInfoV7:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewCustomTypedActorExtractorMap(
				map[cid.Cid][]actorstate.ActorStateExtractor{
					saminer7.Actor{}.Code(): {minertask.V7SectorInfoExtractor{}},
				},
			))

			//
			// Power
			//
		case tasktype.PowerActorClaim:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				power.AllCodes(),
				powertask.ClaimedPowerExtractor{},
			))
		case tasktype.ChainPower:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				power.AllCodes(),
				powertask.ChainPowerExtractor{},
			))

			//
			// Reward
			//
		case tasktype.ChainReward:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				reward.AllCodes(),
				rewardtask.RewardExtractor{},
			))

			//
			// Init
			//
		case tasktype.IdAddress:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				init_.AllCodes(),
				inittask.InitExtractor{},
			))

			//
			// Market
			//
		case tasktype.MarketDealState:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				market.AllCodes(),
				markettask.DealStateExtractor{},
			))
		case tasktype.MarketDealProposal:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				market.AllCodes(),
				markettask.DealProposalExtractor{},
			))

			//
			// Multisig
			//
		case tasktype.MultisigTransaction:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(
				multisig.AllCodes(),
				multisigtask.MultiSigActorExtractor{},
			))

			//
			// Verified Registry
			//
		case tasktype.VerifiedRegistryVerifier:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(),
				verifregtask.VerifierExtractor{},
			))
		case tasktype.VerifiedRegistryVerifiedClient:
			out.ActorProcessors[t] = actorstate.NewTask(api, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(),
				verifregtask.ClientExtractor{},
			))

			//
			// Raw Actors
			//
		case tasktype.Actor:
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&rawtask.RawActorExtractor{})
			out.ActorProcessors[t] = actorstate.NewTask(api, rae)
		case tasktype.ActorState:
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&rawtask.RawActorStateExtractor{})
			out.ActorProcessors[t] = actorstate.NewTask(api, rae)

			//
			// Messages
			//
		case tasktype.Message:
			out.TipsetsProcessors[t] = message.NewTask(api)
		case tasktype.GasOutputs:
			out.TipsetsProcessors[t] = gasoutput.NewTask(api)
		case tasktype.BlockMessage:
			out.TipsetsProcessors[t] = blockmessage.NewTask(api)
		case tasktype.ParsedMessage:
			out.TipsetsProcessors[t] = parsedmessage.NewTask(api)
		case tasktype.Receipt:
			out.TipsetsProcessors[t] = receipt.NewTask(api)
		case tasktype.InternalMessage:
			out.TipsetsProcessors[t] = internalmessage.NewTask(api)
		case tasktype.InternalParsedMessage:
			out.TipsetsProcessors[t] = internalparsedmessage.NewTask(api)
		case tasktype.MessageGasEconomy:
			out.TipsetsProcessors[t] = gaseconomy.NewTask(api)
		case tasktype.MultisigApproval:
			out.TipsetsProcessors[t] = msapprovals.NewTask(api)

			//
			// Blocks
			//
		case tasktype.BlockHeader:
			out.TipsetProcessors[t] = headers.NewTask()
		case tasktype.BlockParent:
			out.TipsetProcessors[t] = parents.NewTask()
		case tasktype.DrandBlockEntrie:
			out.TipsetProcessors[t] = drand.NewTask()

		case tasktype.ChainEconomics:
			out.TipsetProcessors[t] = chaineconomics.NewTask(api)
		case tasktype.ChainConsensus:
			out.TipsetProcessors[t] = consensus.NewTask(api)

		case BuiltinTaskName:
			out.ReportProcessors[t] = indexer.NewTask(api)
		default:
			return nil, fmt.Errorf("unknown task: %s", t)
		}
	}
	return out, nil
}

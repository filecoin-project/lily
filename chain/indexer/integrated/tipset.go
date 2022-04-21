package integrated

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/types"

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
	"github.com/filecoin-project/lily/chain/indexer/integrated/processor"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	taskapi "github.com/filecoin-project/lily/tasks"
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
	indexTask "github.com/filecoin-project/lily/tasks/indexer"
	"github.com/filecoin-project/lily/tasks/messageexecutions/internal_message"
	"github.com/filecoin-project/lily/tasks/messageexecutions/internal_parsed_message"
	"github.com/filecoin-project/lily/tasks/messages/block_message"
	"github.com/filecoin-project/lily/tasks/messages/gas_economy"
	"github.com/filecoin-project/lily/tasks/messages/gas_output"
	"github.com/filecoin-project/lily/tasks/messages/message"
	"github.com/filecoin-project/lily/tasks/messages/parsed_message"
	"github.com/filecoin-project/lily/tasks/messages/receipt"
	"github.com/filecoin-project/lily/tasks/msapprovals"
)

// TipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
type TipSetIndexer struct {
	name      string
	node      taskapi.DataSource
	taskNames []string

	procBuilder *processor.Builder
}

func (ti *TipSetIndexer) init() error {
	var indexerTasks []string
	for _, taskName := range ti.taskNames {
		if tables, found := tasktype.TaskLookup[taskName]; found {
			// if this is a task look up its corresponding tables
			indexerTasks = append(indexerTasks, tables...)
		} else if _, found := tasktype.TableLookup[taskName]; found {
			// it's not a task, maybe it's a table, if it is added to task list, else this is an unknown task
			indexerTasks = append(indexerTasks, taskName)
		} else {
			return xerrors.Errorf("unknown task: %s", taskName)
		}
	}

	tipsetProcessors := map[string]processor.TipSetProcessor{}
	tipsetsProcessors := map[string]processor.TipSetsProcessor{}
	actorProcessors := map[string]processor.ActorProcessor{}
	reportProcessors := map[string]processor.ReportProcessor{
		"builtin": indexTask.NewTask(ti.node),
	}

	for _, t := range indexerTasks {
		switch t {
		//
		// miners
		//
		case tasktype.MinerCurrentDeadlineInfo:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.DeadlineInfoExtractor{},
			))
		case tasktype.MinerFeeDebt:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.FeeDebtExtractor{},
			))
		case tasktype.MinerInfo:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.InfoExtractor{},
			))
		case tasktype.MinerLockedFund:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.InfoExtractor{},
			))
		case tasktype.MinerPreCommitInfo:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.PreCommitInfoExtractor{},
			))
		case tasktype.MinerSectorDeal:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.SectorDealsExtractor{},
			))
		case tasktype.MinerSectorEvent:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.SectorEventsExtractor{},
			))
		case tasktype.MinerSectorPost:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), minertask.PoStExtractor{},
			))
		case tasktype.MinerSectorInfoV1_6:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewCustomTypedActorExtractorMap(
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
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewCustomTypedActorExtractorMap(
				map[cid.Cid][]actorstate.ActorStateExtractor{
					saminer7.Actor{}.Code(): {minertask.V7SectorInfoExtractor{}},
				},
			))

			//
			// Power
			//
		case tasktype.PowerActorClaim:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				power.AllCodes(),
				powertask.ClaimedPowerExtractor{},
			))
		case tasktype.ChainPower:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				power.AllCodes(),
				powertask.ChainPowerExtractor{},
			))

			//
			// Reward
			//
		case tasktype.ChainReward:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				reward.AllCodes(),
				rewardtask.RewardExtractor{},
			))

			//
			// Init
			//
		case tasktype.IdAddress:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				init_.AllCodes(),
				inittask.InitExtractor{},
			))

			//
			// Market
			//
		case tasktype.MarketDealState:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				market.AllCodes(),
				markettask.DealStateExtractor{},
			))
		case tasktype.MarketDealProposal:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				market.AllCodes(),
				markettask.DealProposalExtractor{},
			))

			//
			// Multisig
			//
		case tasktype.MultisigTransaction:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(
				multisig.AllCodes(),
				multisigtask.MultiSigActorExtractor{},
			))

			//
			// Verified Registry
			//
		case tasktype.VerifiedRegistryVerifier:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(),
				verifregtask.VerifierExtractor{},
			))
		case tasktype.VerifiedRegistryVerifiedClient:
			actorProcessors[t] = actorstate.NewTask(ti.node, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(),
				verifregtask.ClientExtractor{},
			))

			//
			// Raw Actors
			//
		case tasktype.Actor:
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&rawtask.RawActorExtractor{})
			actorProcessors[t] = actorstate.NewTask(ti.node, rae)
		case tasktype.ActorState:
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&rawtask.RawActorStateExtractor{})
			actorProcessors[t] = actorstate.NewTask(ti.node, rae)

			//
			// Messages
			//
		case tasktype.Message:
			tipsetsProcessors[t] = message.NewTask(ti.node)
		case tasktype.GasOutputs:
			tipsetsProcessors[t] = gas_output.NewTask(ti.node)
		case tasktype.BlockMessage:
			tipsetsProcessors[t] = block_message.NewTask(ti.node)
		case tasktype.ParsedMessage:
			tipsetsProcessors[t] = parsed_message.NewTask(ti.node)
		case tasktype.Receipt:
			tipsetsProcessors[t] = receipt.NewTask(ti.node)
		case tasktype.InternalMessage:
			tipsetsProcessors[t] = internal_message.NewTask(ti.node)
		case tasktype.InternalParsedMessage:
			tipsetsProcessors[t] = internal_parsed_message.NewTask(ti.node)
		case tasktype.MessageGasEconomy:
			tipsetsProcessors[t] = gas_economy.NewTask(ti.node)
		case tasktype.MultisigApproval:
			tipsetsProcessors[t] = msapprovals.NewTask(ti.node)

			//
			// Blocks
			//
		case tasktype.BlockHeader:
			tipsetProcessors[t] = headers.NewTask()
		case tasktype.BlockParent:
			tipsetProcessors[t] = parents.NewTask()
		case tasktype.DrandBlockEntrie:
			tipsetProcessors[t] = drand.NewTask()

		case tasktype.ChainEconomics:
			tipsetProcessors[t] = chaineconomics.NewTask(ti.node)
		case tasktype.ChainConsensus:
			tipsetProcessors[t] = consensus.NewTask(ti.node)

		default:
			return xerrors.Errorf("unknown task: %s", t)
		}
	}

	ti.procBuilder = processor.NewBuilder(ti.node, ti.name).
		WithTipSetProcessors(tipsetProcessors).
		WithTipSetsProcessors(tipsetsProcessors).
		WithActorProcessors(actorProcessors).
		WithBuiltinProcessors(reportProcessors)

	return nil
}

type Result struct {
	// Name of the task executed.
	Name string
	// Data extracted during task execution.
	Data model.Persistable
	// Report containing details of task execution success and duration.
	Report visormodel.ProcessingReportList
}

// TipSet keeps no internal state and asynchronously indexes `ts` returning Result's as they extracted.
// If the TipSetIndexer encounters an error (fails to fetch ts's parent) it returns immediately and performs no work.
// If one of the TipSetIndexer's tasks encounters a fatal error, the error is return on the error channel.
func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) (chan *Result, chan error, error) {
	start := time.Now()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Name, t.name))
	ctx, span := otel.Tracer("").Start(ctx, "TipSetIndexer.TipSet")

	if ts.Height() == 0 {
		// bail, the parent of genesis is itself, there is no diff
		return nil, nil, nil
	}

	var executed, current *types.TipSet
	pts, err := t.node.TipSet(ctx, ts.Parents())
	if err != nil {
		return nil, nil, err
	}
	current = ts
	executed = pts

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current_tipset", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed_tipset", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("name", t.name),
			attribute.StringSlice("tasks", t.taskNames),
		)
	}

	log.Infow("index", "reporter", t.name, "current", current.Height(), "executed", executed.Height())
	stateResults, taskNames := t.procBuilder.Build().State(ctx, current, executed)

	// build list of executing tasks, used below to label incomplete tasks as skipped.
	executingTasks := make(map[string]bool, len(taskNames))
	for _, name := range taskNames {
		executingTasks[name] = false
	}

	var (
		outCh = make(chan *Result, len(stateResults))
		errCh = make(chan error)
	)
	go func() {
		defer func() {
			close(outCh)
			defer span.End()
		}()

		for res := range stateResults {
			select {
			// canceled, we ran out of time. Mark incomplete work as skipped and exit.
			case <-ctx.Done():
				for name, complete := range executingTasks {
					if complete {
						continue
					}
					stats.Record(ctx, metrics.TipSetSkip.M(1))
					outCh <- &Result{
						Name: name,
						Data: nil,
						Report: visormodel.ProcessingReportList{
							&visormodel.ProcessingReport{
								Height:            int64(current.Height()),
								StateRoot:         current.ParentState().String(),
								Reporter:          t.name,
								Task:              name,
								StartedAt:         start,
								CompletedAt:       time.Now(),
								Status:            visormodel.ProcessingStatusSkip,
								StatusInformation: "indexer timeout",
							}},
					}
				}
				return
				// received a result
			default:

				llt := log.With("height", current.Height(), "task", res.Task, "reporter", t.name)

				// Was there a fatal error?
				if res.Error != nil {
					llt.Errorw("task returned with error", "error", res.Error.Error())
					errCh <- res.Error
					return
				}
				// processor is complete if we receive a result
				executingTasks[res.Task] = true

				if res.Report == nil || len(res.Report) == 0 {
					// Nothing was done for this tipset
					llt.Debugw("task returned with no report")
					continue
				}

				for idx := range res.Report {
					// Fill in some report metadata
					res.Report[idx].Reporter = t.name
					res.Report[idx].Task = res.Task
					res.Report[idx].StartedAt = res.StartedAt
					res.Report[idx].CompletedAt = res.CompletedAt

					if err := res.Report[idx].ErrorsDetected; err != nil {
						// because error is just an interface it may hold a value of any concrete type that implements it, and if
						// said type has unexported fields json marshaling will fail when persisting.
						e, ok := err.(error)
						if ok {
							res.Report[idx].ErrorsDetected = &struct {
								Error string
							}{Error: e.Error()}
						}
						res.Report[idx].Status = visormodel.ProcessingStatusError
					} else if res.Report[idx].StatusInformation != "" {
						res.Report[idx].Status = visormodel.ProcessingStatusInfo
					} else {
						res.Report[idx].Status = visormodel.ProcessingStatusOK
					}

					llt.Debugw("task report", "status", res.Report[idx].Status, "duration", res.Report[idx].CompletedAt.Sub(res.Report[idx].StartedAt))
				}

				// Persist the processing report and the data in a single transaction
				outCh <- &Result{
					Name:   res.Task,
					Data:   res.Data,
					Report: res.Report,
				}
			}
		}
	}()

	return outCh, errCh, nil
}

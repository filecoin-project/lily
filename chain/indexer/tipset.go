package indexer

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	saminer1 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	saminer2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	saminer3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	saminer4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	saminer5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	saminer6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	saminer7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/chain/indexer/processor"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	init_2 "github.com/filecoin-project/lily/tasks/actorstate/init_"
	market2 "github.com/filecoin-project/lily/tasks/actorstate/market"
	miner2 "github.com/filecoin-project/lily/tasks/actorstate/miner"
	multisig2 "github.com/filecoin-project/lily/tasks/actorstate/multisig"
	power2 "github.com/filecoin-project/lily/tasks/actorstate/power"
	"github.com/filecoin-project/lily/tasks/actorstate/raw"
	reward2 "github.com/filecoin-project/lily/tasks/actorstate/reward"
	verifreg2 "github.com/filecoin-project/lily/tasks/actorstate/verifreg"
	"github.com/filecoin-project/lily/tasks/blocks"
	"github.com/filecoin-project/lily/tasks/chaineconomics"
	"github.com/filecoin-project/lily/tasks/consensus"
	"github.com/filecoin-project/lily/tasks/indexer"
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

var tsLog = logging.Logger("lily/index/tipset")

// TipSetIndexer waits for tipsets and persists their block data into a database.
type TipSetIndexer struct {
	name  string
	node  tasks.DataSource
	tasks []string

	procBuilder *processor.Builder
}

type TipSetIndexerOpt func(t *TipSetIndexer)

// NewTipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
func NewTipSetIndexer(node tasks.DataSource, name string, tasks []string, options ...TipSetIndexerOpt) (*TipSetIndexer, error) {
	tsi := &TipSetIndexer{
		name:  name,
		node:  node,
		tasks: tasks,
	}

	tipsetProcessors := map[string]processor.TipSetProcessor{}
	tipsetsProcessors := map[string]processor.TipSetsProcessor{}
	actorProcessors := map[string]processor.ActorProcessor{}
	reportProcessors := map[string]processor.ReportProcessor{
		"builtin": indexer.NewTask(node),
	}

	for _, t := range tasks {
		switch t {
		//
		// miners
		//
		case MinerCurrentDeadlineInfoTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.DeadlineInfoExtractor{},
			))
		case MinerFeeDebtTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.FeeDebtExtractor{},
			))
		case MinerInfoTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.InfoExtractor{},
			))
		case MinerLockedFundsTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.InfoExtractor{},
			))
		case MinerPreCommitInfoTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.PreCommitInfoExtractor{},
			))
		case MinerSectorDealTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.SectorDealsExtractor{},
			))
		case MinerSectorEventsTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.SectorEventsExtractor{},
			))
		case MinerSectorPoStTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				miner.AllCodes(), miner2.PoStExtractor{},
			))
		case MinerSectorInfoTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewCustomTypedActorExtractorMap(
				map[cid.Cid][]actorstate.ActorStateExtractor{
					saminer1.Actor{}.Code(): {miner2.SectorInfoExtractor{}},
					saminer2.Actor{}.Code(): {miner2.SectorInfoExtractor{}},
					saminer3.Actor{}.Code(): {miner2.SectorInfoExtractor{}},
					saminer4.Actor{}.Code(): {miner2.SectorInfoExtractor{}},
					saminer5.Actor{}.Code(): {miner2.SectorInfoExtractor{}},
					saminer6.Actor{}.Code(): {miner2.SectorInfoExtractor{}},
					saminer7.Actor{}.Code(): {miner2.V7SectorInfoExtractor{}},
				},
			))

			//
			// Power
			//
		case PowerActorClaimTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				power.AllCodes(),
				power2.ClaimedPowerExtractor{},
			))
		case ChainPowersTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				power.AllCodes(),
				power2.ChainPowerExtractor{},
			))

			//
			// Reward
			//
		case ChainRewardsTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				reward.AllCodes(),
				reward2.RewardExtractor{},
			))

			//
			// Init
			//
		case IdAddressTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				init_.AllCodes(),
				init_2.InitExtractor{},
			))

			//
			// Market
			//
		case MarketDealStatesTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				market.AllCodes(),
				market2.DealStateExtractor{},
			))
		case MarketDealProposalsTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				market.AllCodes(),
				market2.DealProposalExtractor{},
			))

			//
			// Multisig
			//
		case MultiSigTransactionTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(
				multisig.AllCodes(),
				multisig2.MultiSigActorExtractor{},
			))

			//
			// Verified Registry
			//
		case VerifiedRegistryVerifierTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(),
				verifreg2.VerifierExtractor{},
			))
		case VerifiedRegistryClientTask:
			actorProcessors[t] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(),
				verifreg2.ClientExtractor{},
			))

			//
			// Raw Actors
			//
		case ActorRawTask:
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&raw.RawActorExtractor{})
			actorProcessors[t] = actorstate.NewTask(node, rae)
		case ActorStatesRawTask:
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&raw.RawActorStateExtractor{})
			actorProcessors[t] = actorstate.NewTask(node, rae)

		case MessagesTask:
			tipsetsProcessors[t] = message.NewTask(node)
		case DerivedGasOutputsTask:
			tipsetsProcessors[t] = gas_output.NewTask(node)
		case BlockMessagesTask:
			tipsetsProcessors[t] = block_message.NewTask(node)
		case ParsedMessageTask:
			tipsetsProcessors[t] = parsed_message.NewTask(node)
		case ReceiptTask:
			tipsetsProcessors[t] = receipt.NewTask(node)
		case InternalMessagesTask:
			tipsetsProcessors[t] = internal_message.NewTask(node)
		case InternalParsedMessagesTask:
			tipsetsProcessors[t] = internal_parsed_message.NewTask(node)
		case MessageGasEconomyTask:
			tipsetsProcessors[t] = gas_economy.NewTask(node)

		case MultiSigApprovalTask:
			tipsetsProcessors[t] = msapprovals.NewTask(node)

		case BlocksTask:
			tipsetProcessors[BlocksTask] = blocks.NewTask()
		case ChainEconomicsTask:
			tipsetProcessors[ChainEconomicsTask] = chaineconomics.NewTask(node)
		case ChainConsensusTask:
			tipsetProcessors[ChainConsensusTask] = consensus.NewTask(node)

		default:
			return nil, xerrors.Errorf("unknown task: %s", t)
		}
	}

	tsi.procBuilder = processor.NewBuilder(node, name).
		WithTipSetProcessors(tipsetProcessors).
		WithTipSetsProcessors(tipsetsProcessors).
		WithActorProcessors(actorProcessors).
		WithBuiltinProcessors(reportProcessors)

	for _, opt := range options {
		opt(tsi)
	}

	return tsi, nil
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
			attribute.StringSlice("tasks", t.tasks),
		)
	}

	tsLog.Infow("index", "current", current.Height(), "executed", executed.Height())
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

				llt := tsLog.With("height", current.Height(), "task", res.Task)

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

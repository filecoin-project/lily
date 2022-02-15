package chain

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
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
	"github.com/filecoin-project/lily/lens/task"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lily/tasks/blocks"
	"github.com/filecoin-project/lily/tasks/chaineconomics"
	"github.com/filecoin-project/lily/tasks/consensus"
	"github.com/filecoin-project/lily/tasks/messageexecutions"
	"github.com/filecoin-project/lily/tasks/messages"
	"github.com/filecoin-project/lily/tasks/msapprovals"
)

const (
	ActorStatesRawTask      = "actorstatesraw"      // task that only extracts raw actor state
	ActorStatesPowerTask    = "actorstatespower"    // task that only extracts power actor states (but not the raw state)
	ActorStatesRewardTask   = "actorstatesreward"   // task that only extracts reward actor states (but not the raw state)
	ActorStatesMinerTask    = "actorstatesminer"    // task that only extracts miner actor states (but not the raw state)
	ActorStatesInitTask     = "actorstatesinit"     // task that only extracts init actor states (but not the raw state)
	ActorStatesMarketTask   = "actorstatesmarket"   // task that only extracts market actor states (but not the raw state)
	ActorStatesMultisigTask = "actorstatesmultisig" // task that only extracts multisig actor states (but not the raw state)
	ActorStatesVerifreg     = "actorstatesverifreg" // task that only extracts verified registry actor states (but not the raw state)
	BlocksTask              = "blocks"              // task that extracts block data
	MessagesTask            = "messages"            // task that extracts message data
	ChainEconomicsTask      = "chaineconomics"      // task that extracts chain economics data
	MultisigApprovalsTask   = "msapprovals"         // task that extracts multisig actor approvals
	ImplicitMessageTask     = "implicitmessage"     // task that extract implicitly executed messages: cron tick and block reward.
	ChainConsensusTask      = "consensus"
)

var AllTasks = []string{
	ActorStatesRawTask,
	ActorStatesPowerTask,
	ActorStatesRewardTask,
	ActorStatesMinerTask,
	ActorStatesInitTask,
	ActorStatesMarketTask,
	ActorStatesMultisigTask,
	ActorStatesVerifreg,
	BlocksTask,
	MessagesTask,
	ChainEconomicsTask,
	MultisigApprovalsTask,
	ImplicitMessageTask,
	ChainConsensusTask,
}

var log = logging.Logger("lily/chain")

var _ TipSetObserver = (*TipSetIndexer)(nil)

// TipSetIndexer waits for tipsets and persists their block data into a database.
type TipSetIndexer struct {
	window  time.Duration
	storage model.Storage
	name    string
	node    task.TaskAPI
	tasks   []string
	wg      sync.WaitGroup

	processor *StateProcessor
	exporter  *ModelExporter
}

type TipSetIndexerOpt func(t *TipSetIndexer)

// NewTipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
func NewTipSetIndexer(node task.TaskAPI, d model.Storage, window time.Duration, name string, tasks []string, options ...TipSetIndexerOpt) (*TipSetIndexer, error) {
	tsi := &TipSetIndexer{
		storage: d,
		window:  window,
		name:    name,
		node:    node,
		tasks:   tasks,
	}

	tipsetProcessors := map[string]TipSetProcessor{}
	tipsetsProcessors := map[string]TipSetsProcessor{}
	actorProcessors := map[string]ActorProcessor{}

	for _, t := range tasks {
		switch t {
		case BlocksTask:
			tipsetProcessors[BlocksTask] = blocks.NewTask()
		case ChainEconomicsTask:
			tipsetProcessors[ChainEconomicsTask] = chaineconomics.NewTask(node)
		case ChainConsensusTask:
			tipsetProcessors[ChainConsensusTask] = consensus.NewTask(node)

		case ActorStatesRawTask:
			actorProcessors[ActorStatesRawTask] = actorstate.NewTask(node, &actorstate.RawActorExtractorMap{})
		case ActorStatesPowerTask:
			actorProcessors[ActorStatesPowerTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(power.AllCodes()))
		case ActorStatesRewardTask:
			actorProcessors[ActorStatesRewardTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(reward.AllCodes()))
		case ActorStatesMinerTask:
			actorProcessors[ActorStatesMinerTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(miner.AllCodes()))
		case ActorStatesInitTask:
			actorProcessors[ActorStatesInitTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(init_.AllCodes()))
		case ActorStatesMarketTask:
			actorProcessors[ActorStatesMarketTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(market.AllCodes()))
		case ActorStatesMultisigTask:
			actorProcessors[ActorStatesMultisigTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(multisig.AllCodes()))
		case ActorStatesVerifreg:
			actorProcessors[ActorStatesVerifreg] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes()))

		case MessagesTask:
			tipsetsProcessors[MessagesTask] = messages.NewTask(node)
		case MultisigApprovalsTask:
			tipsetsProcessors[MultisigApprovalsTask] = msapprovals.NewTask(node)
		case ImplicitMessageTask:
			tipsetsProcessors[ImplicitMessageTask] = messageexecutions.NewTask(node)
		default:
			return nil, xerrors.Errorf("unknown task: %s", t)
		}
	}

	sp := NewStateProcessorBuilder(node, name).
		WithTipSetProcessors(tipsetProcessors).
		WithTipSetsProcessors(tipsetsProcessors).
		WithActorProcessors(actorProcessors).
		Build()

	for _, opt := range options {
		opt(tsi)
	}

	tsi.processor = sp
	tsi.exporter = NewModelExporter(1)

	return tsi, nil
}

// TipSet is called when a new tipset has been discovered
func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Name, t.name))
	ctx, span := otel.Tracer("").Start(ctx, "TipSetIndexer.Current")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("name", t.name),
			attribute.String("window", t.window.String()),
			attribute.StringSlice("tasks", t.tasks),
		)
	}
	defer span.End()

	if ts.Height() == 0 {
		// bail, the parent of genesis is itself, there is no diff
		return nil
	}

	var executed, current *types.TipSet
	pts, err := t.node.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return err
	}
	current = ts
	executed = pts

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current_tipset", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed_tipset", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
		)
	}

	// TODO this should be controlled by the caller of TipSet
	var cancel func()
	var tctx context.Context // cancellable context for the task
	if t.window > 0 {
		// Do as much indexing as possible in the specified time window (usually one epoch when following head of chain)
		// Anything not completed in that time will be marked as incomplete
		tctx, cancel = context.WithTimeout(ctx, t.window)
	} else {
		// Ensure all goroutines are stopped when we exit
		tctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	start := time.Now()

	modelResults := t.processor.ProcessState(tctx, current, executed)

	if err := t.exporter.Export(tctx, t.storage, modelResults); err != nil {
		return err
	}

	log.Infow("indexed tipsets", "duration", time.Since(start), "executed", int64(executed.Height()), "current", int64(current.Height()), "tipset", ts.Key().String())
	return nil
}

func (t *TipSetIndexer) Close() error {
	return t.exporter.Close()
}

type TipSetProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSet(ctx context.Context, current *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type TipSetsProcessor interface {
	// ProcessTipSets processes sequential tipsts (a parent and a child, or an executed and a current). If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type ActorProcessor interface {
	// ProcessActors processes a set of actors. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessActors(ctx context.Context, current *types.TipSet, executed *types.TipSet, actors task.ActorStateChangeDiff) (model.Persistable, *visormodel.ProcessingReport, error)
}

// Other names could be: SystemProcessor, ReportProcessor, IndexProcessor, this is basically a TipSetProcessor with no models
type BuiltinProcessor interface {
	ProcessTipSet(ctx context.Context, current *types.TipSet) (visormodel.ProcessingReportList, error)
}

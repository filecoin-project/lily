package tasks

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tracing"
	"github.com/filecoin-project/lily/storage"
)

const (
	TypeGapFillTipSet = "tipset:gapfill"
)

type GapFillTipSetPayload struct {
	TipSet       *types.TipSet
	Tasks        []string
	TraceCarrier *tracing.TraceCarrier `json:",omitempty"`
}

// HasTraceCarrier returns true iff payload contains a trace.
func (g *GapFillTipSetPayload) HasTraceCarrier() bool {
	return !(g.TraceCarrier == nil)
}

func NewGapFillTipSetTask(ctx context.Context, ts *types.TipSet, tasks []string) (*asynq.Task, error) {
	payload, err := json.Marshal(GapFillTipSetPayload{TipSet: ts, Tasks: tasks, TraceCarrier: tracing.NewTraceCarrier(trace.SpanFromContext(ctx).SpanContext())})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGapFillTipSet, payload), nil
}

type AsynqGapFillTipSetTaskHandler struct {
	indexer indexer.Indexer
	db      *storage.Database
}

func NewGapFillHandler(indexer indexer.Indexer, db *storage.Database) *AsynqGapFillTipSetTaskHandler {
	return &AsynqGapFillTipSetTaskHandler{indexer: indexer, db: db}
}

func (gh *AsynqGapFillTipSetTaskHandler) HandleGapFillTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p GapFillTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Infow("gap fill tipset", "tipset", p.TipSet.String(), "height", p.TipSet.Height(), "tasks", p.Tasks)

	if p.HasTraceCarrier() {
		if sc := p.TraceCarrier.AsSpanContext(); sc.IsValid() {
			ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
		}
	}

	success, err := gh.indexer.TipSet(ctx, p.TipSet, indexer.WithTasks(p.Tasks))
	if err != nil {
		if strings.Contains(err.Error(), blockstore.ErrNotFound.Error()) {
			// return SkipRetry to prevent the task from being retried since nodes do not contain the block
			// TODO: later, reschedule task in "backfill" queue with lily nodes capable of syncing the required data.
			return xerrors.Errorf("indexing tipset for gap fill tipset.(height) %s.(%d): Error %s : %w", p.TipSet.Key().String(), p.TipSet.Height(), err, asynq.SkipRetry)
		} else {
			return err
		}
	}
	if !success {
		log.Errorw("failed to gap fill task successfully", "height", p.TipSet.Height(), "tipset", p.TipSet.Key().String())
		return xerrors.Errorf("gap filling tipset.(height) %s.(%d)", p.TipSet.Key(), p.TipSet.Height())
	} else {
		if err := gh.db.SetGapsFilled(ctx, int64(p.TipSet.Height()), p.Tasks...); err != nil {
			log.Errorw("failed to mark gap as filled", "error", err)
			return err
		}
	}
	return nil
}

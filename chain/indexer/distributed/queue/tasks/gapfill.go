package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tracing"
	"github.com/filecoin-project/lily/storage"
)

const (
	TypeGapFillTipSet = "tipset:gapfill"
)

type GapFillPayload struct {
	TipSet       *types.TipSet
	Tasks        []string
	TraceCarrier *tracing.TraceCarrier `json:",omitempty"`
}

// Attributes returns a slice of attributes for populating tracing span attributes.
func (g GapFillPayload) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("height", int64(g.TipSet.Height())),
		attribute.String("tipset", g.TipSet.Key().String()),
		attribute.StringSlice("tasks", g.Tasks),
	}
}

// MarshalLogObject implement ObjectMarshaler and allows user-defined types to efficiently add themselves to the
// logging context, and to selectively omit information which shouldn't be
// included in logs (e.g., passwords).
func (g GapFillPayload) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("tipset", g.TipSet.Key().String())
	enc.AddInt64("height", int64(g.TipSet.Height()))
	enc.AddString("tasks", fmt.Sprint(g.Tasks))
	return nil
}

// HasTraceCarrier returns true iff payload contains a trace.
func (g GapFillPayload) HasTraceCarrier() bool {
	return !(g.TraceCarrier == nil)
}

func NewGapFillTask(ctx context.Context, ts *types.TipSet, tasks []string) (*asynq.Task, error) {
	payload, err := json.Marshal(GapFillPayload{TipSet: ts, Tasks: tasks, TraceCarrier: tracing.NewTraceCarrier(trace.SpanFromContext(ctx).SpanContext())})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGapFillTipSet, payload), nil
}

func NewGapFillTipSetProcessor(indexer indexer.Indexer, db *storage.Database) *GapFillTipSetProcessor {
	return &GapFillTipSetProcessor{indexer: indexer, db: db}
}

type GapFillTipSetProcessor struct {
	indexer indexer.Indexer
	db      *storage.Database
}

func (gh *GapFillTipSetProcessor) Type() string {
	return TypeGapFillTipSet
}

func (gh *GapFillTipSetProcessor) TaskHandler() asynq.HandlerFunc {
	th := &gapFillTipSetTaskHandler{
		idx: gh.indexer,
		db:  gh.db,
	}
	return th.HandleGapFillTipSetTask
}

type gapFillTipSetTaskHandler struct {
	idx indexer.Indexer
	db  *storage.Database
}

func (gh *gapFillTipSetTaskHandler) HandleGapFillTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p GapFillPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := t.ResultWriter().TaskID()
	log.Infow("gap fill tipset", "taskID", taskID, zap.Inline(p))

	if p.HasTraceCarrier() {
		if sc := p.TraceCarrier.AsSpanContext(); sc.IsValid() {
			ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
		}
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			span.SetAttributes(attribute.String("taskID", taskID))
			span.SetAttributes(p.Attributes()...)
		}
	}

	success, err := gh.idx.TipSet(ctx, p.TipSet, indexer.WithTasks(p.Tasks))
	if err != nil {
		log.Errorw("failed to index tipset for gap fill", "taskID", taskID, zap.Inline(p), "error", err)
		return err
	}
	if !success {
		log.Errorw("failed to gap fill task successfully", "taskID", taskID, zap.Inline(p))
		return fmt.Errorf("gap filling tipset.(height) %s.(%d) taskID: %s", p.TipSet.Key(), p.TipSet.Height(), taskID)
	} else { // nolint: revive
		if err := gh.db.SetGapsFilled(ctx, int64(p.TipSet.Height()), p.Tasks...); err != nil {
			log.Errorw("failed to mark gap as filled", "taskID", taskID, zap.Inline(p), "error", err)
			return err
		}
	}
	log.Infow("gap fill tipset success", "taskID", taskID, zap.Inline(p))
	return nil
}

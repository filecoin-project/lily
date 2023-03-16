package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tracing"
)

var log = logging.Logger("lily/queue/tasks")

const (
	TypeIndexTipSet = "tipset:index"
)

type IndexTipSetPayload struct {
	TipSet       *types.TipSet
	Tasks        []string
	TraceCarrier *tracing.TraceCarrier `json:",omitempty"`
}

// Attributes returns a slice of attributes for populating tracing span attributes.
func (i IndexTipSetPayload) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("height", int64(i.TipSet.Height())),
		attribute.String("tipset", i.TipSet.Key().String()),
		attribute.StringSlice("tasks", i.Tasks),
	}
}

// MarshalLogObject implement ObjectMarshaler and allows user-defined types to efficiently add themselves to the
// logging context, and to selectively omit information which shouldn't be
// included in logs (e.g., passwords).
func (i IndexTipSetPayload) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("tipset", i.TipSet.Key().String())
	enc.AddInt64("height", int64(i.TipSet.Height()))
	enc.AddString("tasks", fmt.Sprint(i.Tasks))
	return nil
}

// HasTraceCarrier returns true iff payload contains a trace.
func (i IndexTipSetPayload) HasTraceCarrier() bool {
	return !(i.TraceCarrier == nil)
}

func NewIndexTipSetTask(ctx context.Context, ts *types.TipSet, tasks []string) (*asynq.Task, error) {
	payload, err := json.Marshal(IndexTipSetPayload{TipSet: ts, Tasks: tasks, TraceCarrier: tracing.NewTraceCarrier(trace.SpanFromContext(ctx).SpanContext())})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeIndexTipSet, payload), nil
}

func NewIndexHandler(i indexer.Indexer) *TipSetTaskHandler {
	return &TipSetTaskHandler{indexer: i}
}

type TipSetTaskHandler struct {
	indexer indexer.Indexer
}

func (ih *TipSetTaskHandler) Type() string {
	return TypeIndexTipSet
}

func (ih *TipSetTaskHandler) Handler() asynq.HandlerFunc {
	return ih.HandleIndexTipSetTask
}

func (ih *TipSetTaskHandler) HandleIndexTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p IndexTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := t.ResultWriter().TaskID()
	log.Infow("indexing tipset", "taskID", taskID, zap.Inline(p))

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

	success, err := ih.indexer.TipSet(ctx, p.TipSet, indexer.WithTasks(p.Tasks))
	if err != nil {
		log.Errorw("failed to index tipset", "taskID", taskID, zap.Inline(p), "error", err)
		return err
	}
	if !success {
		log.Errorw("failed to index task successfully", "taskID", taskID, "height", p.TipSet.Height(), zap.Inline(p))
		return nil
	}
	log.Infow("index tipset success", "taskID", taskID, zap.Inline(p))
	return nil
}

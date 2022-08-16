package tasks

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ErrorHandler struct{}

func (e *ErrorHandler) HandleError(ctx context.Context, task *asynq.Task, err error) {
	switch task.Type() {
	case TypeIndexTipSet:
		HandleIndexTaskError(ctx, task, err)
	case TypeGapFillTipSet:
		HandleGapFillTaskError(ctx, task, err)
	default:
		log.Errorw("unknown task type", "type", task.Type(), "error", err)
	}
}

func HandleGapFillTaskError(ctx context.Context, task *asynq.Task, err error) {
	var p GapFillPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		log.Errorw("failed to decode task type (developer error?)", "error", err)
		return
	}
	if p.HasTraceCarrier() {
		if sc := p.TraceCarrier.AsSpanContext(); sc.IsValid() {
			ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
			trace.SpanFromContext(ctx).RecordError(err)
		}
	}
	log.Errorw("task failed", zap.Inline(p), "type", task.Type(), "error", err)
}

func HandleIndexTaskError(ctx context.Context, task *asynq.Task, err error) {
	var p IndexTaskPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		log.Errorw("failed to decode task type (developer error?)", "error", err)
		return
	}
	if p.HasTraceCarrier() {
		if sc := p.TraceCarrier.AsSpanContext(); sc.IsValid() {
			ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
			trace.SpanFromContext(ctx).RecordError(err)
		}
	}
	log.Errorw("task failed", zap.Inline(p), "type", task.Type(), "error", err)
}

package messages

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type InternalMessage struct {
	tableName     struct{} `pg:"internal_messages"` // nolint: structcheck
	Height        int64    `pg:",pk,notnull,use_zero"`
	Cid           string   `pg:",pk,notnull"`
	StateRoot     string   `pg:",notnull"`
	SourceMessage string
	From          string `pg:",notnull"`
	To            string `pg:",notnull"`
	Value         string `pg:"type:numeric,notnull"`
	Method        uint64 `pg:",use_zero"`
	ActorName     string `pg:",notnull"`
	ActorFamily   string `pg:",notnull"`
	ExitCode      int64  `pg:",use_zero"`
	GasUsed       int64  `pg:",use_zero"`
}

func (im *InternalMessage) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "internal_messages"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, im)
}

type InternalMessageList []*InternalMessage

func (l InternalMessageList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "InternalMessageList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(l)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "internal_messages"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(l))
	return s.PersistModel(ctx, l)
}

type InternalParsedMessage struct {
	tableName struct{} `pg:"internal_parsed_messages"` // nolint: structcheck
	Height    int64    `pg:",pk,notnull,use_zero"`
	Cid       string   `pg:",pk,notnull"`
	From      string   `pg:",notnull"`
	To        string   `pg:",notnull"`
	Value     string   `pg:"type:numeric,notnull"`
	Method    string   `pg:",use_zero"`
	Params    string   `pg:",type:jsonb"`
}

func (ipm *InternalParsedMessage) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "internal_parsed_messages"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, ipm)
}

type InternalParsedMessageList []*InternalParsedMessage

func (l InternalParsedMessageList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "InternalParsedMessageList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(l)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "internal_parsed_messages"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(l))
	return s.PersistModel(ctx, l)
}

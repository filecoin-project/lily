package messages

import (
	"context"
	"fmt"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type Message struct {
	Height int64  `pg:",pk,notnull,use_zero"`
	Cid    string `pg:",pk,notnull"`

	From       string `pg:",notnull"`
	To         string `pg:",notnull"`
	Value      string `pg:"type:numeric,notnull"`
	GasFeeCap  string `pg:"type:numeric,notnull"`
	GasPremium string `pg:"type:numeric,notnull"`

	GasLimit  int64  `pg:",use_zero"`
	SizeBytes int    `pg:",use_zero"`
	Nonce     uint64 `pg:",use_zero"`
	Method    uint64 `pg:",use_zero"`
}

type MessageV0 struct {
	tableName struct{} `pg:"messages"` // nolint: structcheck
	Height    int64    `pg:",pk,notnull,use_zero"`
	Cid       string   `pg:",pk,notnull"`

	From       string `pg:",notnull"`
	To         string `pg:",notnull"`
	Value      string `pg:",notnull"`
	GasFeeCap  string `pg:",notnull"`
	GasPremium string `pg:",notnull"`

	GasLimit  int64  `pg:",use_zero"`
	SizeBytes int    `pg:",use_zero"`
	Nonce     uint64 `pg:",use_zero"`
	Method    uint64 `pg:",use_zero"`
}

func (m *Message) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if m == nil {
			return (*MessageV0)(nil), true
		}

		return &MessageV0{
			Height:     m.Height,
			Cid:        m.Cid,
			From:       m.From,
			To:         m.To,
			Value:      m.Value,
			GasFeeCap:  m.GasFeeCap,
			GasPremium: m.GasPremium,
			GasLimit:   m.GasLimit,
			SizeBytes:  m.SizeBytes,
			Nonce:      m.Nonce,
			Method:     m.Method,
		}, true
	case 1:
		return m, true
	default:
		return nil, false
	}
}

func (m *Message) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "messages"))

	vm, ok := m.AsVersion(version)
	if !ok {
		return fmt.Errorf("Message not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, vm)
}

type Messages []*Message

func (ms Messages) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(ms) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "Messages.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ms)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "messages"))

	if version.Major != 1 {
		vms := make([]interface{}, 0, len(ms))
		for _, m := range ms {
			vm, ok := m.AsVersion(version)
			if !ok {
				return fmt.Errorf("Message not supported for schema version %s", version)
			}
			vms = append(vms, vm)
		}
		return s.PersistModel(ctx, vms)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, len(ms))
	return s.PersistModel(ctx, ms)
}

package messages

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type ParsedMessage struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"parsed_messages"`

	Height int64  `pg:",pk,notnull,use_zero"`
	Cid    string `pg:",pk,notnull"`
	From   string `pg:",notnull"`
	To     string `pg:",notnull"`
	Value  string `pg:"type:numeric,notnull"`
	Method string `pg:",notnull"`
	Params string `pg:",type:jsonb"`
}

type ParsedMessageV0 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"parsed_messages"`
	Height    int64    `pg:",pk,notnull,use_zero"`
	Cid       string   `pg:",pk,notnull"`
	From      string   `pg:",notnull"`
	To        string   `pg:",notnull"`
	Value     string   `pg:",notnull"`
	Method    string   `pg:",notnull"`
	Params    string   `pg:",type:jsonb,notnull"`
}

func (pm *ParsedMessage) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if pm == nil {
			return (*ParsedMessageV0)(nil), true
		}

		return &ParsedMessageV0{
			Height: pm.Height,
			Cid:    pm.Cid,
			From:   pm.From,
			To:     pm.To,
			Value:  pm.Value,
			Method: pm.Method,
			Params: pm.Params,
		}, true
	case 1:
		return pm, true
	default:
		return nil, false
	}
}

func (pm *ParsedMessage) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "parsed_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	vpm, ok := pm.AsVersion(version)
	if !ok {
		return xerrors.Errorf("ParsedMessage not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, vpm)
}

type ParsedMessages []*ParsedMessage

func (pms ParsedMessages) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(pms) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "ParsedMessages.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(pms)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "parsed_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if version.Major != 1 {
		vpms := make([]interface{}, 0, len(pms))
		for _, m := range pms {
			vpm, ok := m.AsVersion(version)
			if !ok {
				return xerrors.Errorf("ParsedMessage not supported for schema version %s", version)
			}
			vpms = append(vpms, vpm)
		}
		return s.PersistModel(ctx, vpms)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, len(pms))
	return s.PersistModel(ctx, pms)
}

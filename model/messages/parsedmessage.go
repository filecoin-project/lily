package messages

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type ParsedMessage struct {
	Height int64  `pg:",pk,notnull,use_zero"`
	Cid    string `pg:",pk,notnull"`
	From   string `pg:",notnull"`
	To     string `pg:",notnull"`
	Value  string `pg:"type:numeric,notnull"`
	Method string `pg:",notnull"`
	Params string `pg:",type:jsonb"`
}

type ParsedMessageV0 struct {
	tableName struct{} `pg:"parsed_messages"` // nolint: structcheck,unused
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

	return s.PersistModel(ctx, vpm)
}

type ParsedMessages []*ParsedMessage

func (pms ParsedMessages) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(pms) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ParsedMessages.Persist", trace.WithAttributes(label.Int("count", len(pms))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "parsed_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if version.Major != 1 {
		// Support older versions, but in a non-optimal way
		for _, m := range pms {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	return s.PersistModel(ctx, pms)
}

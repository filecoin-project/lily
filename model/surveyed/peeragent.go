package observed

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"time"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
)

type PeerAgent struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"surveyed_peer_agents"`

	// SurveyerPeerID is the peer ID of the node performing the survey
	SurveyerPeerID string `pg:",pk,notnull"`

	// ObservedAt is the time the observation was made
	ObservedAt time.Time `pg:",pk,notnull"`

	// RawAgent is the raw peer agent string
	RawAgent string `pg:",pk,notnull"`

	// NormalizedAgent is a parsed version of peer agent string, stripping out patch versions
	NormalizedAgent string `pg:",notnull"`

	// Count is the number of peers with the associated agent
	Count int64 `pg:",use_zero,notnull"`
}

func (p *PeerAgent) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "surveyed_peer_agents"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, p)
}

type PeerAgentList []*PeerAgent

func (l PeerAgentList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "PeerAgentList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(l)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "surveyed_peer_agents"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, l)
}

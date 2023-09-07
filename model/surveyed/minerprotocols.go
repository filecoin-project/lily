package observed

import (
	"context"
	"time"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type MinerProtocol struct {
	tableName struct{} `pg:"surveyed_miner_protocols"` // nolint: structcheck

	// ObservedAt is the time the observation was made.
	ObservedAt time.Time `pg:",pk,notnull"`

	// MinerID is the address of the miner observed.
	MinerID string `pg:",pk,notnull"`

	// PeerID is the peerID of the miner observed.
	PeerID string

	// Agent is the raw peer agent string of the miner.
	Agent string

	// Protocols is the list of protocols supported by the miner.
	Protocols []string
}

func (m *MinerProtocol) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "surveyed_miner_protocols"))

	return s.PersistModel(ctx, m)
}

type MinerProtocolList []*MinerProtocol

func (m MinerProtocolList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	if len(m) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "MinerProtocolList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(m)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "surveyed_miner_protocols"))

	return s.PersistModel(ctx, m)
}

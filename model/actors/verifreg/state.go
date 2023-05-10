package verifreg

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
)

const (
	Added    = "ADDED"
	Removed  = "REMOVED"
	Modified = "MODIFIED"
)

type VerifiedRegistryVerifier struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk,notnull"`
	Address   string `pg:",pk,notnull"`

	Event   string `pg:",notnull,type:verified_registry_event_type"`
	DataCap string `pg:",notnull,type:numeric"`
}

func (v *VerifiedRegistryVerifier) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "verified_registry_verifier"))

	return s.PersistModel(ctx, v)
}

type VerifiedRegistryVerifiersList []*VerifiedRegistryVerifier

func (v VerifiedRegistryVerifiersList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "verified_registry_verifier"))

	return s.PersistModel(ctx, v)
}

type VerifiedRegistryVerifiedClient struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk,notnull"`
	Address   string `pg:",pk,notnull"`

	Event   string `pg:",notnull,type:verified_registry_event_type"`
	DataCap string `pg:"type:numeric,notnull"`
}

func (v *VerifiedRegistryVerifiedClient) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "verified_registry_verified_client"))

	return s.PersistModel(ctx, v)
}

type VerifiedRegistryVerifiedClientsList []*VerifiedRegistryVerifiedClient

func (v VerifiedRegistryVerifiedClientsList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "verified_registry_verified_client"))

	return s.PersistModel(ctx, v)
}

var _ model.Persistable = (*VerifiedRegistryVerifier)(nil)
var _ model.Persistable = (*VerifiedRegistryVerifiersList)(nil)
var _ model.Persistable = (*VerifiedRegistryVerifiedClient)(nil)
var _ model.Persistable = (*VerifiedRegistryVerifiedClientsList)(nil)

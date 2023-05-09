package verifreg

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type VerifiedRegistryClaim struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk,notnull"`
	ClaimID   uint64 `pg:",pk,notnull"`
	Provider  string `pg:",notnull"`
	Client    string `pg:",notnull"`
	Data      string `pg:",notnull"`
	Size      uint64 `pg:",notnull,use_zero"`
	TermMin   int64  `pg:",notnull,use_zero"`
	TermMax   int64  `pg:",notnull,use_zero"`
	TermStart int64  `pg:",notnull,use_zero"`
	Sector    uint64 `pg:",notnull,use_zero"`
	Event     string `pg:",notnull,type:verified_registry_event_type"`
}

func (v *VerifiedRegistryClaim) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "verified_registry_claim"))

	return s.PersistModel(ctx, v)
}

type VerifiedRegistryClaimList []*VerifiedRegistryClaim

func (v VerifiedRegistryClaimList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(v) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "verified_registry_claim"))

	return s.PersistModel(ctx, v)
}

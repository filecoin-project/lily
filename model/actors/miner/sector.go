package miner

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

// MinerSectorInfoV7 is the default model exported from the miner actor extractor.
// the table is returned iff the miner actor code is greater than or equal to v7.
// The table receives a new name since we cannot rename the miner_sector_info table, else we will break backfill.
type MinerSectorInfoV7 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"miner_sector_infos_v7"`
	Height    int64    `pg:",pk,notnull,use_zero"`
	MinerID   string   `pg:",pk,notnull"`
	SectorID  uint64   `pg:",pk,use_zero"`
	StateRoot string   `pg:",pk,notnull"`

	SealedCID string `pg:",notnull"`

	ActivationEpoch int64 `pg:",use_zero"`
	ExpirationEpoch int64 `pg:",use_zero"`

	DealWeight         string `pg:"type:numeric,notnull"`
	VerifiedDealWeight string `pg:"type:numeric,notnull"`

	InitialPledge         string `pg:"type:numeric,notnull"`
	ExpectedDayReward     string `pg:"type:numeric,notnull"`
	ExpectedStoragePledge string `pg:"type:numeric,notnull"`

	// added in specs-actors v7, will be null for all sectors and only gets set on the first ReplicaUpdate
	SectorKeyCID string
}

// MinerSectorInfoV1_6 is exported from the miner actor iff the actor code is less than v7.
// The table keeps its original name since that's a requirement to support lily backfills
type MinerSectorInfoV1_6 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"miner_sector_infos"`
	Height    int64    `pg:",pk,notnull,use_zero"`
	MinerID   string   `pg:",pk,notnull"`
	SectorID  uint64   `pg:",pk,use_zero"`
	StateRoot string   `pg:",pk,notnull"`

	SealedCID string `pg:",notnull"`

	ActivationEpoch int64 `pg:",use_zero"`
	ExpirationEpoch int64 `pg:",use_zero"`

	DealWeight         string `pg:"type:numeric,notnull"`
	VerifiedDealWeight string `pg:"type:numeric,notnull"`

	InitialPledge         string `pg:"type:numeric,notnull"`
	ExpectedDayReward     string `pg:"type:numeric,notnull"`
	ExpectedStoragePledge string `pg:"type:numeric,notnull"`
}

type MinerSectorInfoV0 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"miner_sector_infos"`
	Height    int64    `pg:",pk,notnull,use_zero"`
	MinerID   string   `pg:",pk,notnull"`
	SectorID  uint64   `pg:",pk,use_zero"`
	StateRoot string   `pg:",pk,notnull"`

	SealedCID string `pg:",notnull"`

	ActivationEpoch int64 `pg:",use_zero"`
	ExpirationEpoch int64 `pg:",use_zero"`

	DealWeight         string `pg:",notnull"`
	VerifiedDealWeight string `pg:",notnull"`

	InitialPledge         string `pg:",notnull"`
	ExpectedDayReward     string `pg:",notnull"`
	ExpectedStoragePledge string `pg:",notnull"`
}

func (msi *MinerSectorInfoV7) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if msi == nil {
			return (*MinerSectorInfoV0)(nil), true
		}

		return &MinerSectorInfoV0{
			Height:                msi.Height,
			MinerID:               msi.MinerID,
			SectorID:              msi.SectorID,
			StateRoot:             msi.StateRoot,
			SealedCID:             msi.SealedCID,
			ActivationEpoch:       msi.ActivationEpoch,
			ExpirationEpoch:       msi.ExpirationEpoch,
			DealWeight:            msi.DealWeight,
			VerifiedDealWeight:    msi.VerifiedDealWeight,
			InitialPledge:         msi.InitialPledge,
			ExpectedDayReward:     msi.ExpectedDayReward,
			ExpectedStoragePledge: msi.ExpectedStoragePledge,
		}, true
	case 1:
		return msi, true
	default:
		return nil, false
	}
}

func (msi *MinerSectorInfoV7) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_infos_v7"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	m, ok := msi.AsVersion(version)
	if !ok {
		return xerrors.Errorf("MinerSectorInfoV7 not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type (
	MinerSectorInfoV7List []*MinerSectorInfoV7
)

func (ml MinerSectorInfoV7List) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerSectorInfoListV7Plus.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ml)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_infos_v7"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}

	if version.Major == 0 {
		// Support older versions, but in a non-optimal way
		for _, m := range ml {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}

func (msi *MinerSectorInfoV1_6) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if msi == nil {
			return (*MinerSectorInfoV0)(nil), true
		}

		return &MinerSectorInfoV0{
			Height:                msi.Height,
			MinerID:               msi.MinerID,
			SectorID:              msi.SectorID,
			StateRoot:             msi.StateRoot,
			SealedCID:             msi.SealedCID,
			ActivationEpoch:       msi.ActivationEpoch,
			ExpirationEpoch:       msi.ExpirationEpoch,
			DealWeight:            msi.DealWeight,
			VerifiedDealWeight:    msi.VerifiedDealWeight,
			InitialPledge:         msi.InitialPledge,
			ExpectedDayReward:     msi.ExpectedDayReward,
			ExpectedStoragePledge: msi.ExpectedStoragePledge,
		}, true
	case 1:
		return msi, true
	default:
		return nil, false
	}
}

func (msi *MinerSectorInfoV1_6) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	m, ok := msi.AsVersion(version)
	if !ok {
		return xerrors.Errorf("MinerSectorInfoV7 not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type MinerSectorInfoV1_6List []*MinerSectorInfoV1_6

func (ml MinerSectorInfoV1_6List) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerSectorInfoV7List.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ml)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}

	if version.Major == 0 {
		// Support older versions, but in a non-optimal way
		for _, m := range ml {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}

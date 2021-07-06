package extractors

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/miner"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/extract"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

func init() {
	extract.Register(&MinerSectorInfo{}, ExtractMinerSectorInfo)
}

func ExtractMinerSectorInfo(ctx context.Context, ec *extract.MinerStateExtractionContext) (model.Persistable, error) {
	sectorChanges := new(miner.SectorChanges)
	sectorModel := MinerSectorInfoList{}
	if !ec.HasPreviousState() {
		msectors, err := ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, err
		}

		sectorChanges.Added = make([]miner.SectorOnChainInfo, len(msectors))
		for idx, sector := range msectors {
			sectorChanges.Added[idx] = *sector
		}
	} else {
		var err error
		sectorChanges, err = extract.GetSectorDiff(ctx, ec)
		if err != nil {
			return nil, err
		}
	}

	for _, added := range sectorChanges.Added {
		sm := &MinerSectorInfo{
			Height:                int64(ec.CurrTs.Height()),
			MinerID:               ec.Address.String(),
			SectorID:              uint64(added.SectorNumber),
			StateRoot:             ec.CurrTs.ParentState().String(),
			SealedCID:             added.SealedCID.String(),
			ActivationEpoch:       int64(added.Activation),
			ExpirationEpoch:       int64(added.Expiration),
			DealWeight:            added.DealWeight.String(),
			VerifiedDealWeight:    added.VerifiedDealWeight.String(),
			InitialPledge:         added.InitialPledge.String(),
			ExpectedDayReward:     added.ExpectedDayReward.String(),
			ExpectedStoragePledge: added.ExpectedStoragePledge.String(),
		}
		sectorModel = append(sectorModel, sm)
	}

	// do the same for extended sectors, since they have a new deadline
	for _, extended := range sectorChanges.Extended {
		sm := &MinerSectorInfo{
			Height:                int64(ec.CurrTs.Height()),
			MinerID:               ec.Address.String(),
			SectorID:              uint64(extended.To.SectorNumber),
			StateRoot:             ec.CurrTs.ParentState().String(),
			SealedCID:             extended.To.SealedCID.String(),
			ActivationEpoch:       int64(extended.To.Activation),
			ExpirationEpoch:       int64(extended.To.Expiration),
			DealWeight:            extended.To.DealWeight.String(),
			VerifiedDealWeight:    extended.To.VerifiedDealWeight.String(),
			InitialPledge:         extended.To.InitialPledge.String(),
			ExpectedDayReward:     extended.To.ExpectedDayReward.String(),
			ExpectedStoragePledge: extended.To.ExpectedStoragePledge.String(),
		}
		sectorModel = append(sectorModel, sm)
	}
	return sectorModel, nil
}

type MinerSectorInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

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

func (msi *MinerSectorInfo) AsVersion(version model.Version) (interface{}, bool) {
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

func (msi *MinerSectorInfo) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	m, ok := msi.AsVersion(version)
	if !ok {
		return xerrors.Errorf("MinerSectorInfo not supported for schema version %s", version)
	}

	return s.PersistModel(ctx, m)
}

type (
	MinerSectorInfoList []*MinerSectorInfo
)

func (ml MinerSectorInfoList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorInfoList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}

	if version.Major != 1 {
		// Support older versions, but in a non-optimal way
		for _, m := range ml {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	return s.PersistModel(ctx, ml)
}

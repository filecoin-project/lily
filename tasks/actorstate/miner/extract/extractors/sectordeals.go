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
)

func init() {
	extract.Register(&MinerSectorDeal{}, ExtractMinerSectorDeals)
}

func ExtractMinerSectorDeals(ctx context.Context, ec *extract.MinerStateExtractionContext) (model.Persistable, error) {
	sectorChanges := new(miner.SectorChanges)
	sectorDealsModel := MinerSectorDealList{}
	if !ec.HasPreviousState() {
		msectors, err := ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, err
		}

		sectorChanges.Added = make([]miner.SectorOnChainInfo, len(msectors))
		for idx, sector := range msectors {
			sectorChanges.Added[idx] = *sector
			for _, dealID := range sector.DealIDs {
				sectorDealsModel = append(sectorDealsModel, &MinerSectorDeal{
					Height:   int64(ec.CurrTs.Height()),
					MinerID:  ec.Address.String(),
					SectorID: uint64(sector.SectorNumber),
					DealID:   uint64(dealID),
				})
			}
		}
	} else {
		var err error
		sectorChanges, err = extract.GetSectorDiff(ctx, ec)
		if err != nil {
			return nil, err
		}
	}

	for _, newSector := range sectorChanges.Added {
		for _, dealID := range newSector.DealIDs {
			sectorDealsModel = append(sectorDealsModel, &MinerSectorDeal{
				Height:   int64(ec.CurrTs.Height()),
				MinerID:  ec.Address.String(),
				SectorID: uint64(newSector.SectorNumber),
				DealID:   uint64(dealID),
			})
		}
	}
	return sectorDealsModel, nil
}

type MinerSectorDeal struct {
	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,use_zero"`
	DealID   uint64 `pg:",pk,use_zero"`
}

func (ds *MinerSectorDeal) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_deals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, ds)
}

type MinerSectorDealList []*MinerSectorDeal

func (ml MinerSectorDealList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorDealList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_deals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}

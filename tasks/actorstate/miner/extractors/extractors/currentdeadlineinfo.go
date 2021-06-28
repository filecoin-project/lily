package extractors

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/extractors"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
)

func init() {
	extractors.Register(&MinerCurrentDeadlineInfo{}, ExtractMinerCurrentDeadlineInfo)
}

func ExtractMinerCurrentDeadlineInfo(ctx context.Context, ec *extractors.MinerStateExtractionContext) (model.Persistable, error) {
	_, span := global.Tracer("").Start(ctx, "ExtractMinerDeadlineInfo")
	defer span.End()
	currDeadlineInfo, err := ec.CurrState.DeadlineInfo(ec.CurrTs.Height())
	if err != nil {
		return nil, err
	}

	if ec.HasPreviousState() {
		prevDeadlineInfo, err := ec.PrevState.DeadlineInfo(ec.CurrTs.Height())
		if err != nil {
			return nil, err
		}
		if prevDeadlineInfo == currDeadlineInfo {
			return nil, nil
		}
	}

	return &MinerCurrentDeadlineInfo{
		Height:        int64(ec.CurrTs.Height()),
		MinerID:       ec.Address.String(),
		StateRoot:     ec.CurrTs.ParentState().String(),
		DeadlineIndex: currDeadlineInfo.Index,
		PeriodStart:   int64(currDeadlineInfo.PeriodStart),
		Open:          int64(currDeadlineInfo.Open),
		Close:         int64(currDeadlineInfo.Close),
		Challenge:     int64(currDeadlineInfo.Challenge),
		FaultCutoff:   int64(currDeadlineInfo.FaultCutoff),
	}, nil
}

type MinerCurrentDeadlineInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	DeadlineIndex uint64 `pg:",notnull,use_zero"`
	PeriodStart   int64  `pg:",notnull,use_zero"`
	Open          int64  `pg:",notnull,use_zero"`
	Close         int64  `pg:",notnull,use_zero"`
	Challenge     int64  `pg:",notnull,use_zero"`
	FaultCutoff   int64  `pg:",notnull,use_zero"`
}

func (m *MinerCurrentDeadlineInfo) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerCurrentDeadlineInfo.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_current_deadline_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, m)
}

type MinerCurrentDeadlineInfoList []*MinerCurrentDeadlineInfo

func (ml MinerCurrentDeadlineInfoList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerCurrentDeadlineInfoList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_current_deadline_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}

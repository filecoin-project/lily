package miner

import (
	"context"

	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MinerInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	OwnerID  string `pg:",notnull"`
	WorkerID string `pg:",notnull"`

	NewWorker         string
	WorkerChangeEpoch int64 `pg:",notnull,use_zero"`

	ConsensusFaultedElapsed int64 `pg:",notnull,use_zero"`

	PeerID           string
	ControlAddresses []string
	MultiAddresses   []string
}

func (m *MinerInfo) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerInfoModel.Persist")
	defer span.End()
	return s.PersistModel(ctx, m)
}

type MinerInfoList []*MinerInfo

func (ml MinerInfoList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerInfoList.Persist")
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}

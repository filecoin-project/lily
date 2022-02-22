package miner

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type PreCommitInfoExtractor struct{}

func (PreCommitInfoExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "PoStExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "PreCommitInfo.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("creating miner state extraction context: %w", err)
	}

	preCommitChanges, err := node.DiffPreCommits(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	preCommitModel := minermodel.MinerPreCommitInfoList{}
	for _, added := range preCommitChanges.Added {
		pcm := &minermodel.MinerPreCommitInfo{
			Height:    int64(ec.CurrTs.Height()),
			MinerID:   a.Address.String(),
			SectorID:  uint64(added.Info.SectorNumber),
			StateRoot: a.Current.ParentState().String(),

			SealedCID:       added.Info.SealedCID.String(),
			SealRandEpoch:   int64(added.Info.SealRandEpoch),
			ExpirationEpoch: int64(added.Info.Expiration),

			PreCommitDeposit:   added.PreCommitDeposit.String(),
			PreCommitEpoch:     int64(added.PreCommitEpoch),
			DealWeight:         added.DealWeight.String(),
			VerifiedDealWeight: added.VerifiedDealWeight.String(),

			IsReplaceCapacity:      added.Info.ReplaceCapacity,
			ReplaceSectorDeadline:  added.Info.ReplaceSectorDeadline,
			ReplaceSectorPartition: added.Info.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(added.Info.ReplaceSectorNumber),
		}
		preCommitModel = append(preCommitModel, pcm)
	}

	return preCommitModel, nil
}

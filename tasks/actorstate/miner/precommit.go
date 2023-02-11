package miner

import (
	"context"
	"fmt"

	minertypes "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

// PreCommitInfoExtractorV8 extracts precommit infos for all miners v8 and previous.
type PreCommitInfoExtractorV8 struct{}

func (PreCommitInfoExtractorV8) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "PreCommitInfoV8Extractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "PreCommitInfo.Transform")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	var preCommits []minertypes.SectorPreCommitOnChainInfo
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachPrecommittedSectorV8(func(info minertypes.SectorPreCommitOnChainInfo) error {
			preCommits = append(preCommits, info)
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		preCommitChanges, err := node.DiffPreCommitsV8(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
		preCommits = append(preCommits, preCommitChanges.Added...)
	}

	preCommitModel := make(minermodel.MinerPreCommitInfoList, len(preCommits))
	for i, preCommit := range preCommits {
		preCommitModel[i] = &minermodel.MinerPreCommitInfo{
			Height:                 int64(ec.CurrTs.Height()),
			MinerID:                a.Address.String(),
			StateRoot:              a.Current.ParentState().String(),
			SectorID:               uint64(preCommit.Info.SectorNumber),
			SealedCID:              preCommit.Info.SealedCID.String(),
			SealRandEpoch:          int64(preCommit.Info.SealRandEpoch),
			ExpirationEpoch:        int64(preCommit.Info.Expiration),
			PreCommitDeposit:       preCommit.PreCommitDeposit.String(),
			PreCommitEpoch:         int64(preCommit.PreCommitEpoch),
			DealWeight:             preCommit.DealWeight.String(),
			VerifiedDealWeight:     preCommit.VerifiedDealWeight.String(),
			IsReplaceCapacity:      preCommit.Info.ReplaceCapacity,
			ReplaceSectorDeadline:  preCommit.Info.ReplaceSectorDeadline,
			ReplaceSectorPartition: preCommit.Info.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(preCommit.Info.ReplaceSectorNumber),
		}
	}

	return preCommitModel, nil
}

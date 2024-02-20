package miner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	minertypes "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type PreCommitInfoExtractorV9 struct{}

func (PreCommitInfoExtractorV9) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "PreCommitInfoV9Extractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "PreCommitInfoV9.Extract")
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
		if err := ec.CurrState.ForEachPrecommittedSector(func(info minertypes.SectorPreCommitOnChainInfo) error {
			preCommits = append(preCommits, info)
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		preCommitChanges, err := node.DiffPreCommits(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
		preCommits = append(preCommits, preCommitChanges.Added...)
	}

	preCommitModel := make(minermodel.MinerPreCommitInfoV9List, len(preCommits))
	for i, preCommit := range preCommits {
		deals := make([]uint64, len(preCommit.Info.DealIDs))
		for didx, deal := range preCommit.Info.DealIDs {
			deals[didx] = uint64(deal)
		}
		unSealedCID := ""
		if preCommit.Info.UnsealedCid != nil {
			unSealedCID = preCommit.Info.UnsealedCid.String()
		}
		preCommitModel[i] = &minermodel.MinerPreCommitInfoV9{
			Height:           int64(a.Current.Height()),
			StateRoot:        a.Current.ParentState().String(),
			MinerID:          a.Address.String(),
			SectorID:         uint64(preCommit.Info.SectorNumber),
			PreCommitDeposit: preCommit.PreCommitDeposit.String(),
			PreCommitEpoch:   int64(preCommit.PreCommitEpoch),
			SealedCID:        preCommit.Info.SealedCID.String(),
			SealRandEpoch:    int64(preCommit.Info.SealRandEpoch),
			ExpirationEpoch:  int64(preCommit.Info.Expiration),
			DealIDS:          deals,
			UnsealedCID:      unSealedCID,
		}
	}

	return preCommitModel, nil
}

func (PreCommitInfoExtractorV9) Transform(_ context.Context, data model.PersistableList) (model.PersistableList, error) {
	persistableList := make(minermodel.MinerPreCommitInfoV9List, 0, len(data))
	for _, d := range data {
		ml, ok := d.(minermodel.MinerPreCommitInfoV9List)
		if !ok {
			return nil, fmt.Errorf("expected MinerPreCommitInfoV9List type but got: %T", d)
		}
		for _, m := range ml {
			persistableList = append(persistableList, m)
		}
	}
	return model.PersistableList{persistableList}, nil
}

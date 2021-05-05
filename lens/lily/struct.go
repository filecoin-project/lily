package lily

import (
	"context"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/schedule"
)

var log = logging.Logger("lily-api")

type LilyAPIStruct struct {
	// authentication
	v0api.CommonStruct

	// chain notifications and inspection
	// actor state extraction
	v0api.FullNodeStruct

	Internal struct {
		Store                                func() adt.Store                                                                                            `perm:"read"`
		GetExecutedAndBlockMessagesForTipset func(context.Context, *types.TipSet, *types.TipSet) ([]*lens.ExecutedMessage, []*lens.BlockMessages, error) `perm:"read"`

		LilyWatch func(context.Context, *LilyWatchConfig) (schedule.JobID, error) `perm:"read"`
		LilyWalk  func(context.Context, *LilyWalkConfig) (schedule.JobID, error)  `perm:"read"`

		LilyJobStart func(ctx context.Context, ID schedule.JobID) error      `perm:"read"`
		LilyJobStop  func(ctx context.Context, ID schedule.JobID) error      `perm:"read"`
		LilyJobList  func(ctx context.Context) ([]schedule.JobResult, error) `perm:"read"`
	}
}

func (s *LilyAPIStruct) Store() adt.Store {
	return s.Internal.Store()
}

func (s *LilyAPIStruct) LilyWatch(ctx context.Context, cfg *LilyWatchConfig) (schedule.JobID, error) {
	return s.Internal.LilyWatch(ctx, cfg)
}

func (s *LilyAPIStruct) LilyWalk(ctx context.Context, cfg *LilyWalkConfig) (schedule.JobID, error) {
	return s.Internal.LilyWalk(ctx, cfg)
}

func (s *LilyAPIStruct) LilyJobStart(ctx context.Context, ID schedule.JobID) error {
	return s.Internal.LilyJobStart(ctx, ID)
}

func (s *LilyAPIStruct) LilyJobStop(ctx context.Context, ID schedule.JobID) error {
	return s.Internal.LilyJobStop(ctx, ID)
}

func (s *LilyAPIStruct) LilyJobList(ctx context.Context) ([]schedule.JobResult, error) {
	return s.Internal.LilyJobList(ctx)
}

func (s *LilyAPIStruct) GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, []*lens.BlockMessages, error) {
	return s.Internal.GetExecutedAndBlockMessagesForTipset(ctx, ts, pts)
}

var _ LilyAPI = &LilyAPIStruct{}

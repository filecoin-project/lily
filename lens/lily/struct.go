package lily

import (
	"context"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/schedule"
)

var log = logging.Logger("lily-api")

var _ LilyAPI = (*LilyAPIStruct)(nil)

type LilyAPIStruct struct {
	// authentication
	// TODO: avoid importing CommonStruct, split out into separate visor structs
	v0api.CommonStruct

	Internal struct {
		Store                                func() adt.Store                                                                  `perm:"read"`
		GetExecutedAndBlockMessagesForTipset func(context.Context, *types.TipSet, *types.TipSet) (*lens.TipSetMessages, error) `perm:"read"`

		LilyWatch func(context.Context, *LilyWatchConfig) (schedule.JobID, error) `perm:"read"`
		LilyWalk  func(context.Context, *LilyWalkConfig) (schedule.JobID, error)  `perm:"read"`

		LilyJobStart func(ctx context.Context, ID schedule.JobID) error      `perm:"read"`
		LilyJobStop  func(ctx context.Context, ID schedule.JobID) error      `perm:"read"`
		LilyJobList  func(ctx context.Context) ([]schedule.JobResult, error) `perm:"read"`

		SyncState func(ctx context.Context) (*api.SyncState, error) `perm:"read"`
		ChainHead func(context.Context) (*types.TipSet, error)      `perm:"read"`
		ID        func(context.Context) (peer.ID, error)            `perm:"read"`
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

func (s *LilyAPIStruct) GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error) {
	return s.Internal.GetExecutedAndBlockMessagesForTipset(ctx, ts, pts)
}

func (s *LilyAPIStruct) SyncState(ctx context.Context) (*api.SyncState, error) {
	return s.Internal.SyncState(ctx)
}

func (s *LilyAPIStruct) ChainHead(ctx context.Context) (*types.TipSet, error) {
	return s.Internal.ChainHead(ctx)
}

func (s *LilyAPIStruct) ID(ctx context.Context) (peer.ID, error) {
	return s.Internal.ID(ctx)
}

func (s *LilyAPIStruct) SyncState(ctx context.Context) (*api.SyncState, error) {
	return s.Internal.SyncState(ctx)
}

func (s *LilyAPIStruct) ChainHead(ctx context.Context) (*types.TipSet, error) {
	return s.Internal.ChainHead(ctx)
}

var _ LilyAPI = &LilyAPIStruct{}

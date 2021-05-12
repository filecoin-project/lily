package lily

import (
	"context"
	"time"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p-core/peer"

	// "github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/schedule"
)

type LilyAPI interface {
	// NOTE: when adding daemon methods here, don't forget to add to the implementation LilyAPIStruct too

	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error)

	LilyWatch(ctx context.Context, cfg *LilyWatchConfig) (schedule.JobID, error)
	LilyWalk(ctx context.Context, cfg *LilyWalkConfig) (schedule.JobID, error)

	LilyJobStart(ctx context.Context, ID schedule.JobID) error
	LilyJobStop(ctx context.Context, ID schedule.JobID) error
	LilyJobList(ctx context.Context) ([]schedule.JobResult, error)

	// SyncState returns the current status of the chain sync system.
	SyncState(context.Context) (*api.SyncState, error) //perm:read

	ChainHead(context.Context) (*types.TipSet, error) //perm:read

	// ID returns peerID of libp2p node backing this API
	ID(context.Context) (peer.ID, error) //perm:read

}

type LilyWatchConfig struct {
	Name                string
	Tasks               []string
	Window              time.Duration
	Confidence          int
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	Storage             string // name of storage system to use, may be empty
}

type LilyWalkConfig struct {
	From                int64
	To                  int64
	Name                string
	Tasks               []string
	Window              time.Duration
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	Storage             string // name of storage system to use, may be empty
}

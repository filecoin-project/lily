package lily

import (
	"context"
	"time"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/filecoin-project/lily/schedule"
)

type LilyAPI interface {
	// NOTE: when adding daemon methods here, don't forget to add to the implementation to LilyAPIStruct too

	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error)

	LilyIndex(ctx context.Context, cfg *LilyIndexConfig) (interface{}, error)
	LilyWatch(ctx context.Context, cfg *LilyWatchConfig) (*schedule.JobSubmitResult, error)
	LilyWalk(ctx context.Context, cfg *LilyWalkConfig) (*schedule.JobSubmitResult, error)
	LilySurvey(ctx context.Context, cfg *LilySurveyConfig) (*schedule.JobSubmitResult, error)

	LilyIndexNotify(ctx context.Context, cfg *LilyIndexNotifyConfig) (interface{}, error)
	LilyWatchNotify(ctx context.Context, cfg *LilyWatchNotifyConfig) (*schedule.JobSubmitResult, error)
	LilyWalkNotify(ctx context.Context, cfg *LilyWalkNotifyConfig) (*schedule.JobSubmitResult, error)

	LilyJobStart(ctx context.Context, ID schedule.JobID) error
	LilyJobStop(ctx context.Context, ID schedule.JobID) error
	LilyJobWait(ctx context.Context, ID schedule.JobID) (*schedule.JobListResult, error)
	LilyJobList(ctx context.Context) ([]schedule.JobListResult, error)

	LilyGapFind(ctx context.Context, cfg *LilyGapFindConfig) (*schedule.JobSubmitResult, error)
	LilyGapFill(ctx context.Context, cfg *LilyGapFillConfig) (*schedule.JobSubmitResult, error)
	LilyGapFillNotify(ctx context.Context, cfg *LilyGapFillNotifyConfig) (*schedule.JobSubmitResult, error)

	// SyncState returns the current status of the chain sync system.
	SyncState(context.Context) (*api.SyncState, error) //perm:read

	ChainHead(context.Context) (*types.TipSet, error)                                                  //perm:read
	ChainGetBlock(context.Context, cid.Cid) (*types.BlockHeader, error)                                //perm:read
	ChainReadObj(context.Context, cid.Cid) ([]byte, error)                                             //perm:read
	ChainStatObj(context.Context, cid.Cid, cid.Cid) (api.ObjStat, error)                               //perm:read
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)                            //perm:read
	ChainGetTipSetByHeight(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)    //perm:read
	ChainGetTipSetAfterHeight(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error) //perm:read
	ChainGetBlockMessages(context.Context, cid.Cid) (*api.BlockMessages, error)                        //perm:read
	ChainGetParentReceipts(context.Context, cid.Cid) ([]*types.MessageReceipt, error)                  //perm:read
	ChainGetParentMessages(context.Context, cid.Cid) ([]api.Message, error)                            //perm:read
	ChainSetHead(context.Context, types.TipSetKey) error                                               //perm:read
	ChainGetGenesis(context.Context) (*types.TipSet, error)                                            //perm:read

	// trigger graceful shutdown
	Shutdown(context.Context) error

	// LogList returns a list of loggers
	LogList(context.Context) ([]string, error)                       //perm:write
	LogSetLevel(context.Context, string, string) error               //perm:write
	LogSetLevelRegex(ctx context.Context, regex, level string) error //perm:write

	// ID returns peerID of libp2p node backing this API
	ID(context.Context) (peer.ID, error) //perm:read
	NetAutoNatStatus(ctx context.Context) (i api.NatInfo, err error)
	NetPeers(context.Context) ([]peer.AddrInfo, error)
	NetAddrsListen(context.Context) (peer.AddrInfo, error)
	NetPubsubScores(context.Context) ([]api.PubsubScore, error)
	NetAgentVersion(ctx context.Context, p peer.ID) (string, error)
	NetPeerInfo(context.Context, peer.ID) (*api.ExtendedPeerInfo, error)

	StartTipSetWorker(ctx context.Context, cfg *LilyTipSetWorkerConfig) (*schedule.JobSubmitResult, error)

	FindOldestState(ctx context.Context, limit int64) ([]*StateReport, error)
	StateCompute(ctx context.Context, tsk types.TipSetKey) (interface{}, error)
}
type LilyJobConfig struct {
	// Name is the name of the job.
	Name string
	// Tasks are executed by the job.
	Tasks []string
	// Window after which if an execution of the job is not complete it will be canceled.
	Window time.Duration
	// RestartOnFailure when true will restart the job if it encounters an error.
	RestartOnFailure bool
	// RestartOnCompletion when true will restart the job when it completes.
	RestartOnCompletion bool
	// RestartDelay configures how long to wait before restarting the job.
	RestartDelay time.Duration
	// Storage is the name of the storage system the job will use, may be empty.
	Storage string
	// Current Height
	CurrentHeight int
}

type LilyWatchConfig struct {
	JobConfig LilyJobConfig

	BufferSize int // number of tipsets to buffer from notifier service
	Confidence int
	Workers    int // number of indexing jobs that can run in parallel
}

type LilyWatchNotifyConfig struct {
	JobConfig LilyJobConfig

	BufferSize int // number of tipsets to buffer from notifier service
	Confidence int
	Queue      string
}

type LilyWalkConfig struct {
	JobConfig LilyJobConfig

	From int64
	To   int64
}

type LilyWalkNotifyConfig struct {
	WalkConfig LilyWalkConfig

	Queue string
}

type LilyGapFindConfig struct {
	JobConfig LilyJobConfig

	To   int64
	From int64
}

type LilyGapFillConfig struct {
	JobConfig LilyJobConfig

	To   int64
	From int64
}

type LilyGapFillNotifyConfig struct {
	GapFillConfig LilyGapFillConfig

	Queue string
}

type LilyTipSetWorkerConfig struct {
	JobConfig LilyJobConfig

	// Queue is the name of the queueing system the worker will consume work from.
	Queue string
}

type LilySurveyConfig struct {
	JobConfig LilyJobConfig

	Interval time.Duration
}

type LilyIndexConfig struct {
	JobConfig LilyJobConfig

	TipSet types.TipSetKey
}

type LilyIndexNotifyConfig struct {
	IndexConfig LilyIndexConfig

	Queue string
}

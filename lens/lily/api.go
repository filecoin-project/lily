package lily

import (
	"context"
	"time"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"

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
}

type LilyIndexConfig struct {
	TipSet  types.TipSetKey
	Name    string
	Tasks   []string
	Storage string // name of storage system to use, may be empty
	Window  time.Duration
}

type LilyIndexNotifyConfig struct {
	TipSet types.TipSetKey
	Name   string
	Tasks  []string
	Queue  string
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
	Workers             int    // number of indexing jobs that can run in parallel
	BufferSize          int    // number of tipsets to buffer from notifier service
}

type LilyWatchNotifyConfig struct {
	Name                string
	Tasks               []string
	Confidence          int
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	BufferSize          int // number of tipsets to buffer from notifier service
	Queue               string
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
	Workers             int    // number of indexing jobs that can run in parallel
}

type LilyWalkNotifyConfig struct {
	From                int64
	To                  int64
	Name                string
	Tasks               []string
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	Queue               string
}

type LilyGapFindConfig struct {
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	Storage             string // name of storage system to use, cannot be empty and must be Database storage.
	Name                string
	To                  uint64
	From                uint64
	Tasks               []string // name of tasks to fill gaps for
}

type LilyGapFillConfig struct {
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	Storage             string // name of storage system to use, cannot be empty and must be Database storage.
	Name                string
	To                  uint64
	From                uint64
	Tasks               []string // name of tasks to fill gaps for
}

type LilyGapFillNotifyConfig struct {
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	Storage             string // name of storage system to use, cannot be empty and must be Database storage.
	Name                string
	To                  uint64
	From                uint64
	Tasks               []string // name of tasks to fill gaps for
	Queue               string
}

type LilySurveyConfig struct {
	Name                string
	Tasks               []string
	Interval            time.Duration
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
	Storage             string // name of storage system to use, may be empty
}

type LilyTipSetWorkerConfig struct {
	Queue string

	// Concurrency sets the maximum number of concurrent processing of tasks.
	// If set to a zero or negative value, NewServer will overwrite the value
	// to the number of CPUs usable by the current process.
	Concurrency int
	// Storage sets the name of storage system to use, may be empty
	Storage string
	// Name sets the job name
	Name                string
	RestartOnFailure    bool
	RestartOnCompletion bool
	RestartDelay        time.Duration
}

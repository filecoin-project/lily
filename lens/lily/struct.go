package lily

import (
	"context"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/schedule"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
)

var log = logging.Logger("lily/lens")

var _ LilyAPI = (*LilyAPIStruct)(nil)

type LilyAPIStruct struct {
	// authentication
	// TODO: avoid importing CommonStruct, split out into separate lily structs
	v0api.CommonStruct

	Internal struct {
		Store func() adt.Store `perm:"read"`

		LilyIndex  func(ctx context.Context, config *LilyIndexConfig) (interface{}, error)     `perm:"read"`
		LilyWatch  func(context.Context, *LilyWatchConfig) (*schedule.JobSubmitResult, error)  `perm:"read"`
		LilyWalk   func(context.Context, *LilyWalkConfig) (*schedule.JobSubmitResult, error)   `perm:"read"`
		LilySurvey func(context.Context, *LilySurveyConfig) (*schedule.JobSubmitResult, error) `perm:"read"`

		LilyIndexNotify   func(ctx context.Context, config *LilyIndexNotifyConfig) (interface{}, error)                 `perm:"read"`
		LilyWatchNotify   func(ctx context.Context, config *LilyWatchNotifyConfig) (*schedule.JobSubmitResult, error)   `perm:"read"`
		LilyWalkNotify    func(ctx context.Context, config *LilyWalkNotifyConfig) (*schedule.JobSubmitResult, error)    `perm:"read"`
		LilyGapFillNotify func(ctx context.Context, config *LilyGapFillNotifyConfig) (*schedule.JobSubmitResult, error) `perm:"read"`

		LilyJobStart func(ctx context.Context, ID schedule.JobID) error                            `perm:"read"`
		LilyJobStop  func(ctx context.Context, ID schedule.JobID) error                            `perm:"read"`
		LilyJobWait  func(ctx context.Context, ID schedule.JobID) (*schedule.JobListResult, error) `perm:"read"`
		LilyJobList  func(ctx context.Context) ([]schedule.JobListResult, error)                   `perm:"read"`

		LilyGapFind func(ctx context.Context, cfg *LilyGapFindConfig) (*schedule.JobSubmitResult, error) `perm:"read"`
		LilyGapFill func(ctx context.Context, cfg *LilyGapFillConfig) (*schedule.JobSubmitResult, error) `perm:"read"`

		Shutdown func(context.Context) error `perm:"read"`

		SyncState func(ctx context.Context) (*api.SyncState, error) `perm:"read"`

		ChainHead                 func(context.Context) (*types.TipSet, error)                                                    `perm:"read"`
		ChainGetBlock             func(context.Context, cid.Cid) (*types.BlockHeader, error)                                      `perm:"read"`
		ChainReadObj              func(context.Context, cid.Cid) ([]byte, error)                                                  `perm:"read"`
		ChainStatObj              func(context.Context, cid.Cid, cid.Cid) (api.ObjStat, error)                                    `perm:"read"`
		ChainGetTipSet            func(context.Context, types.TipSetKey) (*types.TipSet, error)                                   `perm:"read"`
		ChainGetTipSetByHeight    func(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)                   `perm:"read"`
		ChainGetBlockMessages     func(context.Context, cid.Cid) (*api.BlockMessages, error)                                      `perm:"read"`
		ChainGetParentReceipts    func(context.Context, cid.Cid) ([]*types.MessageReceipt, error)                                 `perm:"read"`
		ChainGetParentMessages    func(context.Context, cid.Cid) ([]api.Message, error)                                           `perm:"read"`
		ChainGetTipSetAfterHeight func(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)                   `perm:"read"`
		ChainSetHead              func(context.Context, types.TipSetKey) error                                                    `perm:"read"`
		ChainGetGenesis           func(context.Context) (*types.TipSet, error)                                                    `perm:"read"`
		ChainPrune                func(ctx context.Context, opts api.PruneOpts) error                                             `perm:"read"`
		ChainHotGC                func(ctx context.Context, opts api.HotGCOpts) error                                             `perm:"read"`
		EthGetBlockByHash         func(ctx context.Context, blkHash ethtypes.EthHash, fullTxInfo bool) (ethtypes.EthBlock, error) `perm:"read"`
		EthGetTransactionReceipt  func(ctx context.Context, txHash ethtypes.EthHash) (*api.EthTxReceipt, error)                   `perm:"read"`
		ChainGetMessagesInTipset  func(ctx context.Context, tsk types.TipSetKey) ([]api.Message, error)                           `perm:"read"`
		EthGetTransactionByHash   func(ctx context.Context, txHash *ethtypes.EthHash) (*ethtypes.EthTx, error)                    `perm:"read"`
		StateListActors           func(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error)                       `perm:"read"`
		GetActorEventsRaw         func(ctx context.Context, filter *types.ActorEventFilter) ([]*types.ActorEvent, error)          `perm:"read"`

		StateCompute func(ctx context.Context, height abi.ChainEpoch, msgs []*types.Message, tsk types.TipSetKey) (*api.ComputeStateOutput, error) `perm:"read"`

		// SyncIncomingBlocks returns a channel streaming incoming, potentially not
		// yet synced block headers.
		SyncIncomingBlocks func(ctx context.Context) (<-chan *types.BlockHeader, error) `perm:"read"`

		LogList          func(context.Context) ([]string, error)     `perm:"read"`
		LogSetLevel      func(context.Context, string, string) error `perm:"read"`
		LogSetLevelRegex func(context.Context, string, string) error `perm:"read"`

		ID               func(context.Context) (peer.ID, error)                        `perm:"read"`
		NetAutoNatStatus func(context.Context) (api.NatInfo, error)                    `perm:"read"`
		NetPeers         func(context.Context) ([]peer.AddrInfo, error)                `perm:"read"`
		NetAddrsListen   func(context.Context) (peer.AddrInfo, error)                  `perm:"read"`
		NetPubsubScores  func(context.Context) ([]api.PubsubScore, error)              `perm:"read"`
		NetAgentVersion  func(context.Context, peer.ID) (string, error)                `perm:"read"`
		NetPeerInfo      func(context.Context, peer.ID) (*api.ExtendedPeerInfo, error) `perm:"read"`
		NetConnect       func(context.Context, peer.AddrInfo) error                    `perm:"read"`
		NetDisconnect    func(context.Context, peer.ID) error                          `perm:"read"`

		StartTipSetWorker func(ctx context.Context, cfg *LilyTipSetWorkerConfig) (*schedule.JobSubmitResult, error) `perm:"read"`

		FindOldestState func(ctx context.Context, limit int64) ([]*StateReport, error)      `perm:"read"`
		RunStateCompute func(ctx context.Context, tsk types.TipSetKey) (interface{}, error) `perm:"read"`
	}
}

func (s *LilyAPIStruct) NetDisconnect(ctx context.Context, id peer.ID) error {
	return s.Internal.NetDisconnect(ctx, id)
}

func (s *LilyAPIStruct) NetConnect(ctx context.Context, p peer.AddrInfo) error {
	return s.Internal.NetConnect(ctx, p)
}

func (s *LilyAPIStruct) RunStateCompute(ctx context.Context, tsk types.TipSetKey) (interface{}, error) {
	return s.Internal.RunStateCompute(ctx, tsk)
}

func (s *LilyAPIStruct) FindOldestState(ctx context.Context, limit int64) ([]*StateReport, error) {
	return s.Internal.FindOldestState(ctx, limit)
}

func (s *LilyAPIStruct) ChainGetTipSetAfterHeight(ctx context.Context, epoch abi.ChainEpoch, key types.TipSetKey) (*types.TipSet, error) {
	return s.Internal.ChainGetTipSetAfterHeight(ctx, epoch, key)
}

func (s *LilyAPIStruct) ChainGetBlock(ctx context.Context, c cid.Cid) (*types.BlockHeader, error) {
	return s.Internal.ChainGetBlock(ctx, c)
}

func (s *LilyAPIStruct) ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error) {
	return s.Internal.ChainReadObj(ctx, c)
}

func (s *LilyAPIStruct) ChainStatObj(ctx context.Context, c cid.Cid, c2 cid.Cid) (api.ObjStat, error) {
	return s.Internal.ChainStatObj(ctx, c, c2)
}

func (s *LilyAPIStruct) ChainGetTipSet(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
	return s.Internal.ChainGetTipSet(ctx, key)
}

func (s *LilyAPIStruct) ChainGetTipSetByHeight(ctx context.Context, epoch abi.ChainEpoch, key types.TipSetKey) (*types.TipSet, error) {
	return s.Internal.ChainGetTipSetByHeight(ctx, epoch, key)
}

func (s *LilyAPIStruct) ChainGetBlockMessages(ctx context.Context, blockCid cid.Cid) (*api.BlockMessages, error) {
	return s.Internal.ChainGetBlockMessages(ctx, blockCid)
}

func (s *LilyAPIStruct) ChainGetParentReceipts(ctx context.Context, blockCid cid.Cid) ([]*types.MessageReceipt, error) {
	return s.Internal.ChainGetParentReceipts(ctx, blockCid)
}

func (s *LilyAPIStruct) ChainGetParentMessages(ctx context.Context, blockCid cid.Cid) ([]api.Message, error) {
	return s.Internal.ChainGetParentMessages(ctx, blockCid)
}

func (s *LilyAPIStruct) ChainSetHead(ctx context.Context, key types.TipSetKey) error {
	return s.Internal.ChainSetHead(ctx, key)
}

func (s *LilyAPIStruct) ChainPrune(ctx context.Context, opts api.PruneOpts) error {
	return s.Internal.ChainPrune(ctx, opts)
}

func (s *LilyAPIStruct) ChainHotGC(ctx context.Context, opts api.HotGCOpts) error {
	return s.Internal.ChainHotGC(ctx, opts)
}

func (s *LilyAPIStruct) ChainGetGenesis(ctx context.Context) (*types.TipSet, error) {
	return s.Internal.ChainGetGenesis(ctx)
}

func (s *LilyAPIStruct) Store() adt.Store {
	return s.Internal.Store()
}

func (s *LilyAPIStruct) LilyIndex(ctx context.Context, cfg *LilyIndexConfig) (interface{}, error) {
	return s.Internal.LilyIndex(ctx, cfg)
}

func (s *LilyAPIStruct) LilyWatch(ctx context.Context, cfg *LilyWatchConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilyWatch(ctx, cfg)
}

func (s *LilyAPIStruct) LilyWalk(ctx context.Context, cfg *LilyWalkConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilyWalk(ctx, cfg)
}

func (s *LilyAPIStruct) LilySurvey(ctx context.Context, cfg *LilySurveyConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilySurvey(ctx, cfg)
}

func (s *LilyAPIStruct) LilyJobStart(ctx context.Context, ID schedule.JobID) error {
	return s.Internal.LilyJobStart(ctx, ID)
}

func (s *LilyAPIStruct) LilyJobStop(ctx context.Context, ID schedule.JobID) error {
	return s.Internal.LilyJobStop(ctx, ID)
}

func (s *LilyAPIStruct) LilyJobWait(ctx context.Context, ID schedule.JobID) (*schedule.JobListResult, error) {
	return s.Internal.LilyJobWait(ctx, ID)
}

func (s *LilyAPIStruct) LilyJobList(ctx context.Context) ([]schedule.JobListResult, error) {
	return s.Internal.LilyJobList(ctx)
}

func (s *LilyAPIStruct) LilyGapFind(ctx context.Context, cfg *LilyGapFindConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilyGapFind(ctx, cfg)
}

func (s *LilyAPIStruct) LilyGapFill(ctx context.Context, cfg *LilyGapFillConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilyGapFill(ctx, cfg)
}

func (s *LilyAPIStruct) Shutdown(ctx context.Context) error {
	return s.Internal.Shutdown(ctx)
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

func (s *LilyAPIStruct) LogList(ctx context.Context) ([]string, error) {
	return s.Internal.LogList(ctx)
}

func (s *LilyAPIStruct) LogSetLevel(ctx context.Context, subsystem, level string) error {
	return s.Internal.LogSetLevel(ctx, subsystem, level)
}

func (s *LilyAPIStruct) LogSetLevelRegex(ctx context.Context, regex, level string) error {
	return s.Internal.LogSetLevelRegex(ctx, regex, level)
}

func (s *LilyAPIStruct) NetAutoNatStatus(ctx context.Context) (api.NatInfo, error) {
	return s.Internal.NetAutoNatStatus(ctx)
}

func (s *LilyAPIStruct) NetPeers(ctx context.Context) ([]peer.AddrInfo, error) {
	return s.Internal.NetPeers(ctx)
}

func (s *LilyAPIStruct) NetAddrsListen(ctx context.Context) (peer.AddrInfo, error) {
	return s.Internal.NetAddrsListen(ctx)
}

func (s *LilyAPIStruct) NetPubsubScores(ctx context.Context) ([]api.PubsubScore, error) {
	return s.Internal.NetPubsubScores(ctx)
}

func (s *LilyAPIStruct) NetAgentVersion(ctx context.Context, p peer.ID) (string, error) {
	return s.Internal.NetAgentVersion(ctx, p)
}

func (s *LilyAPIStruct) NetPeerInfo(ctx context.Context, p peer.ID) (*api.ExtendedPeerInfo, error) {
	return s.Internal.NetPeerInfo(ctx, p)
}

func (s *LilyAPIStruct) StartTipSetWorker(ctx context.Context, cfg *LilyTipSetWorkerConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.StartTipSetWorker(ctx, cfg)
}

func (s *LilyAPIStruct) LilyIndexNotify(ctx context.Context, cfg *LilyIndexNotifyConfig) (interface{}, error) {
	return s.Internal.LilyIndexNotify(ctx, cfg)
}

func (s *LilyAPIStruct) LilyWatchNotify(ctx context.Context, cfg *LilyWatchNotifyConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilyWatchNotify(ctx, cfg)
}

func (s *LilyAPIStruct) LilyWalkNotify(ctx context.Context, cfg *LilyWalkNotifyConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilyWalkNotify(ctx, cfg)
}

func (s *LilyAPIStruct) LilyGapFillNotify(ctx context.Context, cfg *LilyGapFillNotifyConfig) (*schedule.JobSubmitResult, error) {
	return s.Internal.LilyGapFillNotify(ctx, cfg)
}

func (s *LilyAPIStruct) EthGetBlockByHash(ctx context.Context, blkHash ethtypes.EthHash, fullTxInfo bool) (ethtypes.EthBlock, error) {
	return s.Internal.EthGetBlockByHash(ctx, blkHash, fullTxInfo)
}

func (s *LilyAPIStruct) EthGetTransactionReceipt(ctx context.Context, txHash ethtypes.EthHash) (*api.EthTxReceipt, error) {
	return s.Internal.EthGetTransactionReceipt(ctx, txHash)
}

func (s *LilyAPIStruct) ChainGetMessagesInTipset(ctx context.Context, tsk types.TipSetKey) ([]api.Message, error) {
	return s.Internal.ChainGetMessagesInTipset(ctx, tsk)
}

func (s *LilyAPIStruct) StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	return s.Internal.StateListActors(ctx, tsk)
}

func (s *LilyAPIStruct) EthGetTransactionByHash(ctx context.Context, txHash *ethtypes.EthHash) (*ethtypes.EthTx, error) {
	return s.Internal.EthGetTransactionByHash(ctx, txHash)
}

func (s *LilyAPIStruct) SyncIncomingBlocks(ctx context.Context) (<-chan *types.BlockHeader, error) {
	return s.Internal.SyncIncomingBlocks(ctx)
}

func (s *LilyAPIStruct) GetActorEventsRaw(ctx context.Context, filter *types.ActorEventFilter) ([]*types.ActorEvent, error) {
	return s.Internal.GetActorEventsRaw(ctx, filter)
}

func (s *LilyAPIStruct) StateCompute(ctx context.Context, height abi.ChainEpoch, msgs []*types.Message, tsk types.TipSetKey) (*api.ComputeStateOutput, error) {
	return s.Internal.StateCompute(ctx, height, msgs, tsk)
}

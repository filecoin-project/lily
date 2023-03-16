package processor

import (
	"testing"

	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/actors/builtin/datacap"
	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	datacaptask "github.com/filecoin-project/lily/tasks/actorstate/datacap"
	"github.com/filecoin-project/lily/tasks/messageexecutions/vm"
	"github.com/filecoin-project/lily/tasks/messages/actorevent"
	"github.com/filecoin-project/lily/tasks/messages/blockmessage"
	"github.com/filecoin-project/lily/tasks/messages/gaseconomy"
	"github.com/filecoin-project/lily/tasks/messages/message"
	"github.com/filecoin-project/lily/tasks/messages/messageparam"
	"github.com/filecoin-project/lily/tasks/messages/receiptreturn"

	"github.com/filecoin-project/lily/tasks/actorstate"
	inittask "github.com/filecoin-project/lily/tasks/actorstate/init_"
	markettask "github.com/filecoin-project/lily/tasks/actorstate/market"
	minertask "github.com/filecoin-project/lily/tasks/actorstate/miner"
	multisigtask "github.com/filecoin-project/lily/tasks/actorstate/multisig"
	powertask "github.com/filecoin-project/lily/tasks/actorstate/power"
	rawtask "github.com/filecoin-project/lily/tasks/actorstate/raw"
	rewardtask "github.com/filecoin-project/lily/tasks/actorstate/reward"
	verifregtask "github.com/filecoin-project/lily/tasks/actorstate/verifreg"

	"github.com/filecoin-project/lily/tasks/blocks/drand"
	"github.com/filecoin-project/lily/tasks/blocks/headers"
	"github.com/filecoin-project/lily/tasks/blocks/parents"
	"github.com/filecoin-project/lily/tasks/chaineconomics"
	"github.com/filecoin-project/lily/tasks/consensus"
	"github.com/filecoin-project/lily/tasks/messageexecutions/internalmessage"
	"github.com/filecoin-project/lily/tasks/messageexecutions/internalparsedmessage"
	"github.com/filecoin-project/lily/tasks/messages/gasoutput"
	"github.com/filecoin-project/lily/tasks/messages/parsedmessage"
	"github.com/filecoin-project/lily/tasks/messages/receipt"
	"github.com/filecoin-project/lily/tasks/msapprovals"
)

func TestNewProcessor(t *testing.T) {
	proc, err := New(nil, t.Name(), tasktype.AllTableTasks)
	require.NoError(t, err)
	require.Equal(t, t.Name(), proc.name)
	require.Len(t, proc.actorProcessors, 24)
	require.Len(t, proc.tipsetProcessors, 9)
	require.Len(t, proc.tipsetsProcessors, 9)
	require.Len(t, proc.builtinProcessors, 1)

	require.Equal(t, gasoutput.NewTask(nil), proc.tipsetsProcessors[tasktype.GasOutputs])
	require.Equal(t, parsedmessage.NewTask(nil), proc.tipsetsProcessors[tasktype.ParsedMessage])
	require.Equal(t, receipt.NewTask(nil), proc.tipsetsProcessors[tasktype.Receipt])
	require.Equal(t, internalmessage.NewTask(nil), proc.tipsetsProcessors[tasktype.InternalMessage])
	require.Equal(t, internalparsedmessage.NewTask(nil), proc.tipsetsProcessors[tasktype.InternalParsedMessage])
	require.Equal(t, msapprovals.NewTask(nil), proc.tipsetsProcessors[tasktype.MultisigApproval])
	require.Equal(t, vm.NewTask(nil), proc.tipsetsProcessors[tasktype.VMMessage])
	require.Equal(t, actorevent.NewTask(nil), proc.tipsetsProcessors[tasktype.ActorEvent])
	require.Equal(t, receiptreturn.NewTask(nil), proc.tipsetsProcessors[tasktype.ReceiptReturn])

	require.Equal(t, message.NewTask(nil), proc.tipsetProcessors[tasktype.Message])
	require.Equal(t, blockmessage.NewTask(nil), proc.tipsetProcessors[tasktype.BlockMessage])
	require.Equal(t, headers.NewTask(), proc.tipsetProcessors[tasktype.BlockHeader])
	require.Equal(t, parents.NewTask(), proc.tipsetProcessors[tasktype.BlockParent])
	require.Equal(t, drand.NewTask(), proc.tipsetProcessors[tasktype.DrandBlockEntrie])
	require.Equal(t, chaineconomics.NewTask(nil), proc.tipsetProcessors[tasktype.ChainEconomics])
	require.Equal(t, consensus.NewTask(nil), proc.tipsetProcessors[tasktype.ChainConsensus])
	require.Equal(t, gaseconomy.NewTask(nil), proc.tipsetProcessors[tasktype.MessageGasEconomy])
	require.Equal(t, messageparam.NewTask(nil), proc.tipsetProcessors[tasktype.MessageParam])

	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.DeadlineInfoExtractor{})), proc.actorProcessors[tasktype.MinerCurrentDeadlineInfo])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.FeeDebtExtractor{})), proc.actorProcessors[tasktype.MinerFeeDebt])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.InfoExtractor{})), proc.actorProcessors[tasktype.MinerInfo])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.LockedFundsExtractor{})), proc.actorProcessors[tasktype.MinerLockedFund])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorDealsExtractor{})), proc.actorProcessors[tasktype.MinerSectorDeal])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorEventsExtractor{})), proc.actorProcessors[tasktype.MinerSectorEvent])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.PoStExtractor{})), proc.actorProcessors[tasktype.MinerSectorPost])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewCustomTypedActorExtractorMap(
		map[cid.Cid][]actorstate.ActorStateExtractor{
			miner.VersionCodes()[actorstypes.Version0]: {minertask.SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version2]: {minertask.SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version3]: {minertask.SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version4]: {minertask.SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version5]: {minertask.SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version6]: {minertask.SectorInfoExtractor{}},
		},
	)), proc.actorProcessors[tasktype.MinerSectorInfoV1_6])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewCustomTypedActorExtractorMap(
		map[cid.Cid][]actorstate.ActorStateExtractor{
			miner.VersionCodes()[actorstypes.Version7]:  {minertask.V7SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version8]:  {minertask.V7SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version9]:  {minertask.V7SectorInfoExtractor{}},
			miner.VersionCodes()[actorstypes.Version10]: {minertask.V7SectorInfoExtractor{}},
		},
	)), proc.actorProcessors[tasktype.MinerSectorInfoV7])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(power.AllCodes(), powertask.ClaimedPowerExtractor{})), proc.actorProcessors[tasktype.PowerActorClaim])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(power.AllCodes(), powertask.ChainPowerExtractor{})), proc.actorProcessors[tasktype.ChainPower])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(reward.AllCodes(), rewardtask.RewardExtractor{})), proc.actorProcessors[tasktype.ChainReward])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(init_.AllCodes(), inittask.InitExtractor{})), proc.actorProcessors[tasktype.IDAddress])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(market.AllCodes(), markettask.DealStateExtractor{})), proc.actorProcessors[tasktype.MarketDealState])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(market.AllCodes(), markettask.DealProposalExtractor{})), proc.actorProcessors[tasktype.MarketDealProposal])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(multisig.AllCodes(), multisigtask.MultiSigActorExtractor{})), proc.actorProcessors[tasktype.MultisigTransaction])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(), verifregtask.VerifierExtractor{})), proc.actorProcessors[tasktype.VerifiedRegistryVerifier])

	require.Equal(t, actorstate.NewTask(nil, actorstate.NewCustomTypedActorExtractorMap(
		map[cid.Cid][]actorstate.ActorStateExtractor{
			miner.VersionCodes()[actorstypes.Version0]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version2]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version3]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version4]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version5]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version6]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version7]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version8]:  {minertask.PreCommitInfoExtractorV8{}},
			miner.VersionCodes()[actorstypes.Version9]:  {minertask.PreCommitInfoExtractorV9{}},
			miner.VersionCodes()[actorstypes.Version10]: {minertask.PreCommitInfoExtractorV9{}},
		},
	)), proc.actorProcessors[tasktype.MinerPreCommitInfo])

	require.Equal(t, actorstate.NewTask(nil, actorstate.NewCustomTypedActorExtractorMap(
		map[cid.Cid][]actorstate.ActorStateExtractor{
			verifreg.VersionCodes()[actorstypes.Version0]: {verifregtask.ClientExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version2]: {verifregtask.ClientExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version3]: {verifregtask.ClientExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version4]: {verifregtask.ClientExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version5]: {verifregtask.ClientExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version6]: {verifregtask.ClientExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version7]: {verifregtask.ClientExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version8]: {verifregtask.ClientExtractor{}},
		},
	)), proc.actorProcessors[tasktype.VerifiedRegistryVerifiedClient])

	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(datacap.AllCodes(), datacaptask.BalanceExtractor{})), proc.actorProcessors[tasktype.DataCapBalance])

	require.Equal(t, actorstate.NewTask(nil, actorstate.NewCustomTypedActorExtractorMap(
		map[cid.Cid][]actorstate.ActorStateExtractor{
			verifreg.VersionCodes()[actorstypes.Version9]:  {verifregtask.ClaimExtractor{}},
			verifreg.VersionCodes()[actorstypes.Version10]: {verifregtask.ClaimExtractor{}},
		},
	)), proc.actorProcessors[tasktype.VerifiedRegistryClaim])

	rae := &actorstate.RawActorExtractorMap{}
	rae.Register(&rawtask.RawActorExtractor{})
	require.Equal(t, actorstate.NewTask(nil, rae), proc.actorProcessors[tasktype.Actor])
	rae1 := &actorstate.RawActorExtractorMap{}
	rae1.Register(&rawtask.RawActorStateExtractor{})
	require.Equal(t, actorstate.NewTask(nil, rae1), proc.actorProcessors[tasktype.ActorState])
}

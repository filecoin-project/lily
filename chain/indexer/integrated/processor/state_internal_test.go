package processor

import (
	"testing"

	saminer1 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	saminer2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	saminer3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	saminer4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	saminer5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	saminer6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	saminer7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
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
	"github.com/filecoin-project/lily/tasks/messages/blockmessage"
	"github.com/filecoin-project/lily/tasks/messages/gaseconomy"
	"github.com/filecoin-project/lily/tasks/messages/gasoutput"
	"github.com/filecoin-project/lily/tasks/messages/message"
	"github.com/filecoin-project/lily/tasks/messages/parsedmessage"
	"github.com/filecoin-project/lily/tasks/messages/receipt"
	"github.com/filecoin-project/lily/tasks/msapprovals"
)

func TestNewProcessor(t *testing.T) {
	proc, err := New(nil, t.Name(), tasktype.AllTableTasks)
	require.NoError(t, err)
	require.Equal(t, t.Name(), proc.name)
	require.Len(t, proc.actorProcessors, 21)
	require.Len(t, proc.tipsetProcessors, 5)
	require.Len(t, proc.tipsetsProcessors, 9)
	require.Len(t, proc.builtinProcessors, 1)

	require.Equal(t, message.NewTask(nil), proc.tipsetsProcessors[tasktype.Message])
	require.Equal(t, gasoutput.NewTask(nil), proc.tipsetsProcessors[tasktype.GasOutputs])
	require.Equal(t, blockmessage.NewTask(nil), proc.tipsetsProcessors[tasktype.BlockMessage])
	require.Equal(t, parsedmessage.NewTask(nil), proc.tipsetsProcessors[tasktype.ParsedMessage])
	require.Equal(t, receipt.NewTask(nil), proc.tipsetsProcessors[tasktype.Receipt])
	require.Equal(t, internalmessage.NewTask(nil), proc.tipsetsProcessors[tasktype.InternalMessage])
	require.Equal(t, internalparsedmessage.NewTask(nil), proc.tipsetsProcessors[tasktype.InternalParsedMessage])
	require.Equal(t, gaseconomy.NewTask(nil), proc.tipsetsProcessors[tasktype.MessageGasEconomy])
	require.Equal(t, msapprovals.NewTask(nil), proc.tipsetsProcessors[tasktype.MultisigApproval])

	require.Equal(t, headers.NewTask(), proc.tipsetProcessors[tasktype.BlockHeader])
	require.Equal(t, parents.NewTask(), proc.tipsetProcessors[tasktype.BlockParent])
	require.Equal(t, drand.NewTask(), proc.tipsetProcessors[tasktype.DrandBlockEntrie])
	require.Equal(t, chaineconomics.NewTask(nil), proc.tipsetProcessors[tasktype.ChainEconomics])
	require.Equal(t, consensus.NewTask(nil), proc.tipsetProcessors[tasktype.ChainConsensus])

	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.DeadlineInfoExtractor{})), proc.actorProcessors[tasktype.MinerCurrentDeadlineInfo])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.FeeDebtExtractor{})), proc.actorProcessors[tasktype.MinerFeeDebt])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.InfoExtractor{})), proc.actorProcessors[tasktype.MinerInfo])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.LockedFundsExtractor{})), proc.actorProcessors[tasktype.MinerLockedFund])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.PreCommitInfoExtractor{})), proc.actorProcessors[tasktype.MinerPreCommitInfo])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorDealsExtractor{})), proc.actorProcessors[tasktype.MinerSectorDeal])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorEventsExtractor{})), proc.actorProcessors[tasktype.MinerSectorEvent])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.PoStExtractor{})), proc.actorProcessors[tasktype.MinerSectorPost])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewCustomTypedActorExtractorMap(
		map[cid.Cid][]actorstate.ActorStateExtractor{
			saminer1.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
			saminer2.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
			saminer3.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
			saminer4.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
			saminer5.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
			saminer6.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
		},
	)), proc.actorProcessors[tasktype.MinerSectorInfoV1_6])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewCustomTypedActorExtractorMap(
		map[cid.Cid][]actorstate.ActorStateExtractor{
			saminer7.Actor{}.Code(): {minertask.V7SectorInfoExtractor{}},
		},
	)), proc.actorProcessors[tasktype.MinerSectorInfoV7])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(power.AllCodes(), powertask.ClaimedPowerExtractor{})), proc.actorProcessors[tasktype.PowerActorClaim])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(power.AllCodes(), powertask.ChainPowerExtractor{})), proc.actorProcessors[tasktype.ChainPower])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(reward.AllCodes(), rewardtask.RewardExtractor{})), proc.actorProcessors[tasktype.ChainReward])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(init_.AllCodes(), inittask.InitExtractor{})), proc.actorProcessors[tasktype.IdAddress])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(market.AllCodes(), markettask.DealStateExtractor{})), proc.actorProcessors[tasktype.MarketDealState])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(market.AllCodes(), markettask.DealProposalExtractor{})), proc.actorProcessors[tasktype.MarketDealProposal])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(multisig.AllCodes(), multisigtask.MultiSigActorExtractor{})), proc.actorProcessors[tasktype.MultisigTransaction])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(), verifregtask.VerifierExtractor{})), proc.actorProcessors[tasktype.VerifiedRegistryVerifier])
	require.Equal(t, actorstate.NewTask(nil, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(), verifregtask.ClientExtractor{})), proc.actorProcessors[tasktype.VerifiedRegistryVerifiedClient])
	rae := &actorstate.RawActorExtractorMap{}
	rae.Register(&rawtask.RawActorExtractor{})
	require.Equal(t, actorstate.NewTask(nil, rae), proc.actorProcessors[tasktype.Actor])
	rae1 := &actorstate.RawActorExtractorMap{}
	rae1.Register(&rawtask.RawActorStateExtractor{})
	require.Equal(t, actorstate.NewTask(nil, rae1), proc.actorProcessors[tasktype.ActorState])
}

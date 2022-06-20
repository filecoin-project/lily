package processor_test

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
	"github.com/filecoin-project/lily/chain/indexer/integrated/processor"
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
	"github.com/filecoin-project/lily/tasks/indexer"
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

func TestMakeProcessorsActors(t *testing.T) {
	t.Run("miner extractors", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.MinerCurrentDeadlineInfo,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.DeadlineInfoExtractor{}),
			},
			{
				taskName:  tasktype.MinerFeeDebt,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.FeeDebtExtractor{}),
			},
			{
				taskName:  tasktype.MinerInfo,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.InfoExtractor{}),
			},
			{
				taskName:  tasktype.MinerLockedFund,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.LockedFundsExtractor{}),
			},
			{
				taskName:  tasktype.MinerPreCommitInfo,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.PreCommitInfoExtractor{}),
			},
			{
				taskName:  tasktype.MinerSectorDeal,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorDealsExtractor{}),
			},
			{
				taskName:  tasktype.MinerSectorEvent,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorEventsExtractor{}),
			},
			{
				taskName:  tasktype.MinerSectorPost,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.PoStExtractor{}),
			},
			{
				taskName: tasktype.MinerSectorInfoV1_6,
				extractor: actorstate.NewCustomTypedActorExtractorMap(
					map[cid.Cid][]actorstate.ActorStateExtractor{
						saminer1.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
						saminer2.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
						saminer3.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
						saminer4.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
						saminer5.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
						saminer6.Actor{}.Code(): {minertask.SectorInfoExtractor{}},
					},
				),
			},
			{
				taskName: tasktype.MinerSectorInfoV7,
				extractor: actorstate.NewCustomTypedActorExtractorMap(
					map[cid.Cid][]actorstate.ActorStateExtractor{
						saminer7.Actor{}.Code(): {minertask.V7SectorInfoExtractor{}},
					},
				),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTask(nil, tc.extractor), proc.ActorProcessors[tc.taskName])
			})
		}
	})

	t.Run("power extractors", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.PowerActorClaim,
				extractor: actorstate.NewTypedActorExtractorMap(power.AllCodes(), powertask.ClaimedPowerExtractor{}),
			},
			{
				taskName:  tasktype.ChainPower,
				extractor: actorstate.NewTypedActorExtractorMap(power.AllCodes(), powertask.ChainPowerExtractor{}),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTask(nil, tc.extractor), proc.ActorProcessors[tc.taskName])
			})
		}
	})

	t.Run("reward extractors", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.ChainReward,
				extractor: actorstate.NewTypedActorExtractorMap(reward.AllCodes(), rewardtask.RewardExtractor{}),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTask(nil, tc.extractor), proc.ActorProcessors[tc.taskName])
			})
		}
	})

	t.Run("init extractors", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.IdAddress,
				extractor: actorstate.NewTypedActorExtractorMap(init_.AllCodes(), inittask.InitExtractor{}),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTask(nil, tc.extractor), proc.ActorProcessors[tc.taskName])
			})
		}
	})

	t.Run("market extractors", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.MarketDealState,
				extractor: actorstate.NewTypedActorExtractorMap(market.AllCodes(), markettask.DealStateExtractor{}),
			},
			{
				taskName:  tasktype.MarketDealProposal,
				extractor: actorstate.NewTypedActorExtractorMap(market.AllCodes(), markettask.DealProposalExtractor{}),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTask(nil, tc.extractor), proc.ActorProcessors[tc.taskName])
			})
		}
	})

	t.Run("multisig extractors", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.MultisigTransaction,
				extractor: actorstate.NewTypedActorExtractorMap(multisig.AllCodes(), multisigtask.MultiSigActorExtractor{}),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTask(nil, tc.extractor), proc.ActorProcessors[tc.taskName])
			})
		}
	})

	t.Run("verified registry extractors", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.VerifiedRegistryVerifier,
				extractor: actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(), verifregtask.VerifierExtractor{}),
			},
			{
				taskName:  tasktype.VerifiedRegistryVerifiedClient,
				extractor: actorstate.NewTypedActorExtractorMap(verifreg.AllCodes(), verifregtask.ClientExtractor{}),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTask(nil, tc.extractor), proc.ActorProcessors[tc.taskName])
			})
		}
	})

	t.Run("raw actors", func(t *testing.T) {
		t.Run("actor", func(t *testing.T) {
			proc, err := processor.MakeProcessors(nil, []string{tasktype.Actor})
			require.NoError(t, err)
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&rawtask.RawActorExtractor{})
			require.Len(t, proc.ActorProcessors, 1)
			require.Equal(t, actorstate.NewTask(nil, rae), proc.ActorProcessors[tasktype.Actor])

		})

		t.Run("actor state", func(t *testing.T) {
			proc, err := processor.MakeProcessors(nil, []string{tasktype.ActorState})
			require.NoError(t, err)
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&rawtask.RawActorStateExtractor{})
			require.Len(t, proc.ActorProcessors, 1)
			require.Equal(t, actorstate.NewTask(nil, rae), proc.ActorProcessors[tasktype.ActorState])
		})
	})
}

func TestMakeProcessorsTipSet(t *testing.T) {
	tasks := []string{
		tasktype.BlockHeader,
		tasktype.BlockParent,
		tasktype.DrandBlockEntrie,
		tasktype.ChainEconomics,
		tasktype.ChainConsensus,
	}
	proc, err := processor.MakeProcessors(nil, tasks)
	require.NoError(t, err)
	require.Len(t, proc.TipsetProcessors, len(tasks))

	require.Equal(t, headers.NewTask(), proc.TipsetProcessors[tasktype.BlockHeader])
	require.Equal(t, parents.NewTask(), proc.TipsetProcessors[tasktype.BlockParent])
	require.Equal(t, drand.NewTask(), proc.TipsetProcessors[tasktype.DrandBlockEntrie])
	require.Equal(t, chaineconomics.NewTask(nil), proc.TipsetProcessors[tasktype.ChainEconomics])
	require.Equal(t, consensus.NewTask(nil), proc.TipsetProcessors[tasktype.ChainConsensus])
}

func TestMakeProcessorsTipSets(t *testing.T) {
	tasks := []string{
		tasktype.Message,
		tasktype.GasOutputs,
		tasktype.BlockMessage,
		tasktype.ParsedMessage,
		tasktype.Receipt,
		tasktype.InternalMessage,
		tasktype.InternalParsedMessage,
		tasktype.MessageGasEconomy,
		tasktype.MultisigApproval,
	}
	proc, err := processor.MakeProcessors(nil, tasks)
	require.NoError(t, err)
	require.Len(t, proc.TipsetsProcessors, len(tasks))

	require.Equal(t, message.NewTask(nil), proc.TipsetsProcessors[tasktype.Message])
	require.Equal(t, gasoutput.NewTask(nil), proc.TipsetsProcessors[tasktype.GasOutputs])
	require.Equal(t, blockmessage.NewTask(nil), proc.TipsetsProcessors[tasktype.BlockMessage])
	require.Equal(t, parsedmessage.NewTask(nil), proc.TipsetsProcessors[tasktype.ParsedMessage])
	require.Equal(t, receipt.NewTask(nil), proc.TipsetsProcessors[tasktype.Receipt])
	require.Equal(t, internalmessage.NewTask(nil), proc.TipsetsProcessors[tasktype.InternalMessage])
	require.Equal(t, internalparsedmessage.NewTask(nil), proc.TipsetsProcessors[tasktype.InternalParsedMessage])
	require.Equal(t, gaseconomy.NewTask(nil), proc.TipsetsProcessors[tasktype.MessageGasEconomy])
	require.Equal(t, msapprovals.NewTask(nil), proc.TipsetsProcessors[tasktype.MultisigApproval])
}

func TestMakeProcessorsReport(t *testing.T) {
	proc, err := processor.MakeProcessors(nil, []string{processor.BuiltinTaskName})
	require.NoError(t, err)
	require.Len(t, proc.ReportProcessors, 1)
	require.Equal(t, indexer.NewTask(nil), proc.ReportProcessors[processor.BuiltinTaskName])
}

func TestMakeProcessorsInvalidTaskName(t *testing.T) {
	t.Run("single invalid name", func(t *testing.T) {
		invalidTask := "invalid_task_name"
		proc, err := processor.MakeProcessors(nil, []string{invalidTask})
		require.Error(t, err)
		require.Nil(t, proc)
	})

	t.Run("mix of invalid and valid", func(t *testing.T) {
		validName := tasktype.Message
		invalidName := "invalid_task_name"
		proc, err := processor.MakeProcessors(nil, []string{validName, invalidName})
		require.Error(t, err)
		require.Nil(t, proc)

		proc, err = processor.MakeProcessors(nil, []string{invalidName, validName})
		require.Error(t, err)
		require.Nil(t, proc)
	})
}

func TestMakeProcessorsAllTasks(t *testing.T) {
	// If this test fails it indicates a new processor and/or task name was added and test should be created for it in one of the above test cases.
	proc, err := processor.MakeProcessors(nil, append(tasktype.AllTableTasks, processor.BuiltinTaskName))
	require.NoError(t, err)
	require.Len(t, proc.ActorProcessors, 21)
	require.Len(t, proc.TipsetProcessors, 5)
	require.Len(t, proc.TipsetsProcessors, 9)
	require.Len(t, proc.ReportProcessors, 1)
}

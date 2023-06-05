package processor_test

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
	"github.com/filecoin-project/lily/chain/indexer/integrated/processor"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/tasks/actorstate"
	datacaptask "github.com/filecoin-project/lily/tasks/actorstate/datacap"
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
				taskName: tasktype.MinerPreCommitInfo,
				extractor: actorstate.NewCustomTypedActorExtractorMap(
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
						miner.VersionCodes()[actorstypes.Version11]: {minertask.PreCommitInfoExtractorV9{}},
					},
				),
			},
			{
				taskName:  tasktype.MinerSectorDeal,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorDealsExtractor{}),
			},
			{
				taskName:  tasktype.MinerSectorPost,
				extractor: actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.PoStExtractor{}),
			},
			{
				taskName: tasktype.MinerSectorInfoV1_6,
				extractor: actorstate.NewCustomTypedActorExtractorMap(
					map[cid.Cid][]actorstate.ActorStateExtractor{
						miner.VersionCodes()[actorstypes.Version0]: {minertask.SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version2]: {minertask.SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version3]: {minertask.SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version4]: {minertask.SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version5]: {minertask.SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version6]: {minertask.SectorInfoExtractor{}},
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
		testCases2 := []struct {
			taskName    string
			extractor   actorstate.ActorExtractorMap
			transformer actorstate.ActorDataTransformer
		}{
			{
				taskName:    tasktype.MinerLockedFund,
				extractor:   actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.LockedFundsExtractor{}),
				transformer: minertask.LockedFundsExtractor{},
			},
			{
				taskName:    tasktype.MinerSectorEvent,
				extractor:   actorstate.NewTypedActorExtractorMap(miner.AllCodes(), minertask.SectorEventsExtractor{}),
				transformer: minertask.SectorEventsExtractor{},
			},
			{
				taskName: tasktype.MinerSectorInfoV7,
				extractor: actorstate.NewCustomTypedActorExtractorMap(
					map[cid.Cid][]actorstate.ActorStateExtractor{
						miner.VersionCodes()[actorstypes.Version7]:  {minertask.V7SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version8]:  {minertask.V7SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version9]:  {minertask.V7SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version10]: {minertask.V7SectorInfoExtractor{}},
						miner.VersionCodes()[actorstypes.Version11]: {minertask.V7SectorInfoExtractor{}},
					},
				),
				transformer: minertask.V7SectorInfoExtractor{},
			},
		}
		for _, tc := range testCases2 {
			t.Run(tc.taskName, func(t *testing.T) {
				proc, err := processor.MakeProcessors(nil, []string{tc.taskName})
				require.NoError(t, err)
				require.Len(t, proc.ActorProcessors, 1)
				require.Equal(t, actorstate.NewTaskWithTransformer(nil, tc.extractor, tc.transformer), proc.ActorProcessors[tc.taskName])
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
				taskName:  tasktype.IDAddress,
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
				taskName: tasktype.VerifiedRegistryVerifiedClient,
				extractor: actorstate.NewCustomTypedActorExtractorMap(
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
				),
			},
			{
				taskName: tasktype.VerifiedRegistryClaim,
				extractor: actorstate.NewCustomTypedActorExtractorMap(
					map[cid.Cid][]actorstate.ActorStateExtractor{
						verifreg.VersionCodes()[actorstypes.Version9]:  {verifregtask.ClaimExtractor{}},
						verifreg.VersionCodes()[actorstypes.Version10]: {verifregtask.ClaimExtractor{}},
						verifreg.VersionCodes()[actorstypes.Version11]: {verifregtask.ClaimExtractor{}},
					}),
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

	t.Run("datacap actor extractor", func(t *testing.T) {
		testCases := []struct {
			taskName  string
			extractor actorstate.ActorExtractorMap
		}{
			{
				taskName:  tasktype.DataCapBalance,
				extractor: actorstate.NewTypedActorExtractorMap(datacap.AllCodes(), datacaptask.BalanceExtractor{}),
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
			rat := &rawtask.RawActorExtractor{}
			require.Equal(t, actorstate.NewTaskWithTransformer(nil, rae, rat), proc.ActorProcessors[tasktype.Actor])

		})

		t.Run("actor state", func(t *testing.T) {
			proc, err := processor.MakeProcessors(nil, []string{tasktype.ActorState})
			require.NoError(t, err)
			rae := &actorstate.RawActorExtractorMap{}
			rae.Register(&rawtask.RawActorStateExtractor{})
			require.Len(t, proc.ActorProcessors, 1)
			rat := &rawtask.RawActorStateExtractor{}
			require.Equal(t, actorstate.NewTaskWithTransformer(nil, rae, rat), proc.ActorProcessors[tasktype.ActorState])
		})
	})
}

func TestMakeProcessorsTipSet(t *testing.T) {
	tasks := []string{
		tasktype.Message,
		tasktype.BlockMessage,
		tasktype.BlockHeader,
		tasktype.BlockParent,
		tasktype.DrandBlockEntrie,
		tasktype.ChainEconomics,
		tasktype.ChainConsensus,
		tasktype.MessageGasEconomy,
	}
	proc, err := processor.MakeProcessors(nil, tasks)
	require.NoError(t, err)
	require.Len(t, proc.TipsetProcessors, len(tasks))

	require.Equal(t, message.NewTask(nil), proc.TipsetProcessors[tasktype.Message])
	require.Equal(t, blockmessage.NewTask(nil), proc.TipsetProcessors[tasktype.BlockMessage])
	require.Equal(t, headers.NewTask(), proc.TipsetProcessors[tasktype.BlockHeader])
	require.Equal(t, parents.NewTask(), proc.TipsetProcessors[tasktype.BlockParent])
	require.Equal(t, drand.NewTask(), proc.TipsetProcessors[tasktype.DrandBlockEntrie])
	require.Equal(t, chaineconomics.NewTask(nil), proc.TipsetProcessors[tasktype.ChainEconomics])
	require.Equal(t, consensus.NewTask(nil), proc.TipsetProcessors[tasktype.ChainConsensus])
	require.Equal(t, gaseconomy.NewTask(nil), proc.TipsetProcessors[tasktype.MessageGasEconomy])
}

func TestMakeProcessorsTipSets(t *testing.T) {
	tasks := []string{
		tasktype.GasOutputs,
		tasktype.ParsedMessage,
		tasktype.Receipt,
		tasktype.InternalMessage,
		tasktype.InternalParsedMessage,
		tasktype.MultisigApproval,
	}
	proc, err := processor.MakeProcessors(nil, tasks)
	require.NoError(t, err)
	require.Len(t, proc.TipsetsProcessors, len(tasks))

	require.Equal(t, gasoutput.NewTask(nil), proc.TipsetsProcessors[tasktype.GasOutputs])
	require.Equal(t, parsedmessage.NewTask(nil), proc.TipsetsProcessors[tasktype.ParsedMessage])
	require.Equal(t, receipt.NewTask(nil), proc.TipsetsProcessors[tasktype.Receipt])
	require.Equal(t, internalmessage.NewTask(nil), proc.TipsetsProcessors[tasktype.InternalMessage])
	require.Equal(t, internalparsedmessage.NewTask(nil), proc.TipsetsProcessors[tasktype.InternalParsedMessage])
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
	require.Len(t, proc.ActorProcessors, 24)
	require.Len(t, proc.TipsetProcessors, 12)
	require.Len(t, proc.TipsetsProcessors, 11)
	require.Len(t, proc.ReportProcessors, 1)
}

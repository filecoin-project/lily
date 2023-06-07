package tasktype_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/storage"
)

func TestMakeTaskNamesAlias(t *testing.T) {
	testCases := []struct {
		taskAlias string
		tasks     []string
	}{
		{
			taskAlias: tasktype.ActorStatesRawTask,
			tasks:     []string{tasktype.Actor, tasktype.ActorState},
		},
		{
			taskAlias: tasktype.ActorStatesPowerTask,
			tasks:     []string{tasktype.ChainPower, tasktype.PowerActorClaim},
		},
		{
			taskAlias: tasktype.ActorStatesRewardTask,
			tasks:     []string{tasktype.ChainReward},
		},
		{
			taskAlias: tasktype.ActorStatesMinerTask,
			tasks: []string{tasktype.MinerSectorDeal, tasktype.MinerSectorInfoV7, tasktype.MinerSectorInfoV1_6,
				tasktype.MinerSectorPost, tasktype.MinerPreCommitInfo, tasktype.MinerSectorEvent,
				tasktype.MinerCurrentDeadlineInfo, tasktype.MinerFeeDebt, tasktype.MinerLockedFund, tasktype.MinerInfo,
				tasktype.MinerBeneficiary},
		},
		{
			taskAlias: tasktype.ActorStatesInitTask,
			tasks:     []string{tasktype.IDAddress},
		},
		{
			taskAlias: tasktype.ActorStatesMarketTask,
			tasks:     []string{tasktype.MarketDealProposal, tasktype.MarketDealState},
		},
		{
			taskAlias: tasktype.ActorStatesMultisigTask,
			tasks:     []string{tasktype.MultisigTransaction},
		},
		{
			taskAlias: tasktype.ActorStatesVerifreg,
			tasks:     []string{tasktype.VerifiedRegistryVerifier, tasktype.VerifiedRegistryVerifiedClient, tasktype.DataCapBalance, tasktype.VerifiedRegistryClaim},
		},
		{
			taskAlias: tasktype.BlocksTask,
			tasks:     []string{tasktype.BlockHeader, tasktype.BlockParent, tasktype.DrandBlockEntrie},
		},
		{
			taskAlias: tasktype.MessagesTask,
			tasks:     []string{tasktype.Message, tasktype.ParsedMessage, tasktype.Receipt, tasktype.GasOutputs, tasktype.MessageGasEconomy, tasktype.BlockMessage, tasktype.ActorEvent, tasktype.MessageParam, tasktype.ReceiptReturn},
		},
		{
			taskAlias: tasktype.ChainEconomicsTask,
			tasks:     []string{tasktype.ChainEconomics},
		},
		{
			taskAlias: tasktype.MultisigApprovalsTask,
			tasks:     []string{tasktype.MultisigApproval},
		},
		{
			taskAlias: tasktype.ImplicitMessageTask,
			tasks:     []string{tasktype.InternalMessage, tasktype.InternalParsedMessage, tasktype.VMMessage},
		},
		{
			taskAlias: tasktype.ChainConsensusTask,
			tasks:     []string{tasktype.ChainConsensus},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.taskAlias, func(t *testing.T) {
			actual, err := tasktype.MakeTaskNames([]string{tc.taskAlias})
			require.NoError(t, err)
			require.Len(t, actual, len(tc.tasks))
			for _, task := range tc.tasks {
				require.Contains(t, actual, task)
			}
		})
	}
}

func TestMakeAllTaskAliasNames(t *testing.T) {
	var taskAliases []string
	for alias := range tasktype.TaskLookup {
		taskAliases = append(taskAliases, alias)
	}
	actual, err := tasktype.MakeTaskNames(taskAliases)
	require.NoError(t, err)
	// if this test fails it means a new task or new task alias was created, update the above test cases.
	require.Len(t, actual, len(tasktype.AllTableTasks))
	// if this test fails it means a new model was added that doesn't have a task name.
	require.Len(t, actual, len(storage.Models))
}

func TestMakeAllTaskNames(t *testing.T) {
	const TotalTableTasks = 48
	actual, err := tasktype.MakeTaskNames(tasktype.AllTableTasks)
	require.NoError(t, err)
	// if this test fails it means a new task name was added, update the above test
	require.Len(t, actual, TotalTableTasks)
}

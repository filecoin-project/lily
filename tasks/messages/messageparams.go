package messages

import (
	"bytes"
	"fmt"

	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	sa4builtin "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"

	"github.com/filecoin-project/sentinel-visor/tasks/messages/types"
)

type methodMeta struct {
	Name string
	ipld.NodePrototype
}
type methodtable map[int64]methodMeta

var initTable = methodtable{
	2: methodMeta{"InitExecParams", types.Type.MessageParamsInitExecParams__Repr},
}

var marketTable = methodtable{
	2: methodMeta{"AddBalance", types.Type.Address__Repr},
	3: methodMeta{"WithdrawBalance", types.Type.MessageParamsMarketWithdrawBalance__Repr},
	4: methodMeta{"PublishStorageDeals", types.Type.MessageParamsMarketPublishDeals__Repr},
	5: methodMeta{"VerifyDealsForActivation", types.Type.MessageParamsMarketVerifyDeals__Repr},
	6: methodMeta{"ActivateDeals", types.Type.MessageParamsMarketActivateDeals__Repr},
	7: methodMeta{"OnMinerSectorsTerminate", types.Type.MessageParamsMarketTerminateDeals__Repr},
	8: methodMeta{"ComputeDataCommitment", types.Type.MessageParamsMarketComputeCommitment__Repr},
	9: methodMeta{"CronTick", types.Type.Any__Repr},
}

var minerTable = methodtable{
	1:  methodMeta{"Constructor", types.Type.MessageParamsMinerConstructor__Repr},
	2:  methodMeta{"ControlAddresses", types.Type.Any__Repr},
	3:  methodMeta{"ChangeWorkerAddress", types.Type.MessageParamsMinerChangeAddress__Repr},
	4:  methodMeta{"ChangePeerID", types.Type.MessageParamsMinerChangePeerID__Repr},
	5:  methodMeta{"SubmitWindowedPoSt", types.Type.MessageParamsMinerSubmitWindowedPoSt__Repr},
	6:  methodMeta{"PreCommitSector", types.Type.MinerV0SectorPreCommitInfo__Repr},
	7:  methodMeta{"ProveCommitSector", types.Type.MessageParamsMinerProveCommitSector__Repr},
	8:  methodMeta{"ExtendSectorExpiration", types.Type.MessageParamsMinerExtendSectorExpiration__Repr},
	9:  methodMeta{"TerminateSectors", types.Type.MessageParamsMinerTerminateSectors__Repr},
	10: methodMeta{"DeclareFaults", types.Type.MessageParamsMinerDeclareFaults__Repr},
	11: methodMeta{"DeclareFaultsRecovered", types.Type.MessageParamsMinerDeclareFaultsRecovered__Repr},
	12: methodMeta{"OnDeferredCronEvent", types.Type.MessageParamsMinerDeferredCron__Repr},
	13: methodMeta{"CheckSectorProven", types.Type.MessageParamsMinerCheckSectorProven__Repr},
	14: methodMeta{"ApplyRewards", types.Type.ApplyRewardParams__Repr},
	15: methodMeta{"ReportConsensusFault", types.Type.MessageParamsMinerReportFault__Repr},
	16: methodMeta{"WithdrawBalance", types.Type.MessageParamsMinerWithdrawBalance__Repr},
	17: methodMeta{"ConfirmSectorProofsValid", types.Type.MessageParamsMinerConfirmSectorProofs__Repr},
	18: methodMeta{"ChangeMultiaddrs", types.Type.MessageParamsMinerChangeMultiaddrs__Repr},
	19: methodMeta{"CompactPartitions", types.Type.MessageParamsMinerCompactPartitions__Repr},
	20: methodMeta{"CompactSectorNumbers", types.Type.MessageParamsMinerCompactSectorNumbers__Repr},
	21: methodMeta{"ConfirmUpdateWorkerKey", types.Type.Any__Repr},
	22: methodMeta{"RepayDebt", types.Type.Any__Repr},
	23: methodMeta{"ChangeOwnerAddress", types.Type.Address__Repr},
}

var multisigTable = methodtable{
	1: methodMeta{"Constructor", types.Type.MessageParamsMultisigConstructor__Repr},
	2: methodMeta{"Propose", types.Type.MessageParamsMultisigPropose__Repr},
	3: methodMeta{"Approve", types.Type.MessageParamsMultisigTxnID__Repr},
	4: methodMeta{"Cancel", types.Type.MessageParamsMultisigTxnID__Repr},
	5: methodMeta{"AddSigner", types.Type.MessageParamsMultisigAddSigner__Repr},
	6: methodMeta{"RemoveSigner", types.Type.MessageParamsMultisigRemoveSigner__Repr},
	7: methodMeta{"SwapSigner", types.Type.MessageParamsMultisigSwapSigner__Repr},
	8: methodMeta{"ChangeNumApprovalsThreshold", types.Type.MessageParamsMultisigChangeThreshold__Repr},
	9: methodMeta{"LockBalance", types.Type.MessageParamsMultisigLockBalance__Repr},
}

var paychTable = methodtable{
	1: methodMeta{"Constructor", types.Type.MessageParamsPaychConstructor__Repr},
	2: methodMeta{"UpdateChannelState", types.Type.MessageParamsPaychUpdateChannelState__Repr},
	3: methodMeta{"Settle", types.Type.Any__Repr},
	4: methodMeta{"Collect", types.Type.Any__Repr},
}

var powerTable = methodtable{
	1: methodMeta{"Constructor", types.Type.Any__Repr},
	2: methodMeta{"CreateMiner", types.Type.MessageParamsPowerCreateMiner__Repr},
	3: methodMeta{"UpdateClaimedPower", types.Type.MessageParamsPowerUpdateClaimed__Repr},
	4: methodMeta{"EnrollCronEvent", types.Type.MessageParamsPowerEnrollCron__Repr},
	5: methodMeta{"OnEpochTickEnd", types.Type.Any__Repr},
	6: methodMeta{"UpdatePledgeTotal", types.Type.BigInt__Repr},
	7: methodMeta{"Nil", types.Type.Any__Repr}, // deprecated
	8: methodMeta{"SubmitPoRepForBulkVerify", types.Type.SealVerifyInfo__Repr},
	9: methodMeta{"CurrentTotalPower", types.Type.MessageParamsPowerCurrentTotal__Repr},
}

var rewardTable = methodtable{
	1: methodMeta{"Constructor", types.Type.BigInt__Repr},
	2: methodMeta{"AwardBlockRewards", types.Type.MessageParamsRewardAwardBlock__Repr},
	3: methodMeta{"ThisEpochReward", types.Type.Any__Repr},
	4: methodMeta{"UpdateNetworkKPI", types.Type.BigInt__Repr},
}

var verifregTable = methodtable{
	1: methodMeta{"Constructor", types.Type.Address__Repr},
	2: methodMeta{"AddVerifier", types.Type.MessageParamsVerifregAddVerifier__Repr},
	3: methodMeta{"RemoveVerifier", types.Type.Address__Repr},
	4: methodMeta{"AddVerifiedClient", types.Type.MessageParamsVerifregAddVerifier__Repr},
	5: methodMeta{"UseBytes", types.Type.MessageParamsVerifregUseBytes__Repr},
	6: methodMeta{"RestoreBytes", types.Type.MessageParamsVerifregUseBytes__Repr},
}

// LotusType represents known types
type LotusType string

const (
	LotusTypeUnknown             LotusType = "unknown"
	AccountActorState            LotusType = "accountActor"
	CronActorState               LotusType = "cronActor"
	InitActorState               LotusType = "initActor"
	InitActorV3State             LotusType = "initActorV3"
	MarketActorState             LotusType = "storageMarketActor"
	MarketActorV2State           LotusType = "storageMarketActorV2"
	MarketActorV3State           LotusType = "storageMarketActorV3"
	MultisigActorState           LotusType = "multisigActor"
	MultisigActorV3State         LotusType = "multisigActorV3"
	PaymentChannelActorState     LotusType = "paymentChannelActor"
	PaymentChannelActorV3State   LotusType = "paymentChannelActorV3"
	RewardActorState             LotusType = "rewardActor"
	RewardActorV2State           LotusType = "rewardActorV2"
	StorageMinerActorState       LotusType = "storageMinerActor"
	StorageMinerActorV2State     LotusType = "storageMinerActorV2"
	StorageMinerActorV3State     LotusType = "storageMinerActorV3"
	StorageMinerActorV4State     LotusType = "storageMinerActorV4"
	StoragePowerActorState       LotusType = "storagePowerActor"
	StoragePowerActorV2State     LotusType = "storagePowerActorV2"
	StoragePowerActorV3State     LotusType = "storagePowerActorV3"
	VerifiedRegistryActorState   LotusType = "verifiedRegistryActor"
	VerifiedRegistryActorV3State LotusType = "verifiedRegistryActorV3"
)

var messageParamTable = map[LotusType]methodtable{
	InitActorState:               initTable,
	InitActorV3State:             initTable,
	MarketActorState:             marketTable,
	MarketActorV2State:           marketTable,
	MarketActorV3State:           marketTable,
	StorageMinerActorState:       minerTable,
	StorageMinerActorV2State:     minerTable,
	StorageMinerActorV3State:     minerTable,
	StorageMinerActorV4State:     minerTable,
	MultisigActorState:           multisigTable,
	MultisigActorV3State:         multisigTable,
	PaymentChannelActorState:     paychTable,
	PaymentChannelActorV3State:   paychTable,
	StoragePowerActorState:       powerTable,
	StoragePowerActorV2State:     powerTable,
	StoragePowerActorV3State:     powerTable,
	RewardActorState:             rewardTable,
	RewardActorV2State:           rewardTable,
	VerifiedRegistryActorState:   verifregTable,
	VerifiedRegistryActorV3State: verifregTable,
	LotusTypeUnknown:             {},
}

// LotusActorCodes for actor states
var LotusActorCodes = map[cid.Cid]LotusType{
	sa0builtin.AccountActorCodeID:          AccountActorState,
	sa0builtin.CronActorCodeID:             CronActorState,
	sa0builtin.InitActorCodeID:             InitActorState,
	sa0builtin.MultisigActorCodeID:         MultisigActorState,
	sa0builtin.PaymentChannelActorCodeID:   PaymentChannelActorState,
	sa0builtin.RewardActorCodeID:           RewardActorState,
	sa0builtin.StorageMarketActorCodeID:    MarketActorState,
	sa0builtin.StorageMinerActorCodeID:     StorageMinerActorState,
	sa0builtin.StoragePowerActorCodeID:     StoragePowerActorState,
	sa0builtin.SystemActorCodeID:           LotusType("systemActor"),
	sa0builtin.VerifiedRegistryActorCodeID: VerifiedRegistryActorState,

	// v2
	sa2builtin.AccountActorCodeID:          AccountActorState,
	sa2builtin.CronActorCodeID:             CronActorState,
	sa2builtin.InitActorCodeID:             InitActorState,
	sa2builtin.MultisigActorCodeID:         MultisigActorState,
	sa2builtin.PaymentChannelActorCodeID:   PaymentChannelActorState,
	sa2builtin.RewardActorCodeID:           RewardActorV2State,
	sa2builtin.StorageMarketActorCodeID:    MarketActorV2State,
	sa2builtin.StorageMinerActorCodeID:     StorageMinerActorV2State,
	sa2builtin.StoragePowerActorCodeID:     StoragePowerActorV2State,
	sa2builtin.SystemActorCodeID:           LotusType("systemActor"),
	sa2builtin.VerifiedRegistryActorCodeID: VerifiedRegistryActorState,

	// v3
	sa3builtin.AccountActorCodeID:          AccountActorState,
	sa3builtin.CronActorCodeID:             CronActorState,
	sa3builtin.InitActorCodeID:             InitActorV3State,
	sa3builtin.MultisigActorCodeID:         MultisigActorV3State,
	sa3builtin.PaymentChannelActorCodeID:   PaymentChannelActorV3State,
	sa3builtin.RewardActorCodeID:           RewardActorV2State,
	sa3builtin.StorageMarketActorCodeID:    MarketActorV3State,
	sa3builtin.StorageMinerActorCodeID:     StorageMinerActorV3State,
	sa3builtin.StoragePowerActorCodeID:     StoragePowerActorV3State,
	sa3builtin.SystemActorCodeID:           LotusType("systemActor"),
	sa3builtin.VerifiedRegistryActorCodeID: VerifiedRegistryActorV3State,

	// v4
	sa4builtin.AccountActorCodeID:          AccountActorState,
	sa4builtin.CronActorCodeID:             CronActorState,
	sa4builtin.InitActorCodeID:             InitActorV3State,
	sa4builtin.MultisigActorCodeID:         MultisigActorV3State,
	sa4builtin.PaymentChannelActorCodeID:   PaymentChannelActorV3State,
	sa4builtin.RewardActorCodeID:           RewardActorV2State,
	sa4builtin.StorageMarketActorCodeID:    MarketActorV3State,
	sa4builtin.StorageMinerActorCodeID:     StorageMinerActorV4State,
	sa4builtin.StoragePowerActorCodeID:     StoragePowerActorV3State,
	sa4builtin.SystemActorCodeID:           LotusType("systemActor"),
	sa4builtin.VerifiedRegistryActorCodeID: VerifiedRegistryActorV3State,
}

func ParseParams(params []byte, method int64, destType LotusType) (ipld.Node, string, error) {
	mthdTable, ok := messageParamTable[destType]
	if !ok {
		return nil, "", fmt.Errorf("unknown parameters for %s", destType)
	}

	proto := ipld.NodePrototype(types.Type.Any__Repr)
	name := "Unknown"
	mthd, ok := mthdTable[method]
	if ok {
		proto = mthd.NodePrototype
		name = mthd.Name
	}

	if len(params) == 0 {
		b, err := types.Type.Bytes__Repr.FromBytes(params)
		return b, name, err
	}

	builder := proto.NewBuilder()
	if err := dagcbor.Decoder(builder, bytes.NewBuffer(params)); err != nil {
		return nil, "", fmt.Errorf("cbor decode into %s (%s.%d) failed: %v", name, destType, method, err)
	}

	return builder.Build(), name, nil
}

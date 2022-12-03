package tasktype

import (
	"fmt"
	"math"
	"strings"

	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/build"
	"github.com/go-pg/pg/v10/orm"

	"github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/model/actors/datacap"
	init_ "github.com/filecoin-project/lily/model/actors/init"
	"github.com/filecoin-project/lily/model/actors/market"
	"github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/model/actors/multisig"
	"github.com/filecoin-project/lily/model/actors/power"
	"github.com/filecoin-project/lily/model/actors/reward"
	"github.com/filecoin-project/lily/model/actors/verifreg"
	"github.com/filecoin-project/lily/model/blocks"
	"github.com/filecoin-project/lily/model/chain"
	"github.com/filecoin-project/lily/model/derived"
	"github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lily/model/msapprovals"
)

type Table struct {
	// Name is the name of the table
	Name string

	// Task is the name of the task that writes the table
	Task string

	// Schema is the major schema version for which the table is supported.
	Schema int

	// NetworkVersionRange is the range filecoin network versions for which the table is supported.
	NetworkVersionRange NetworkVersionRange

	// An empty instance of the lily model
	Model interface{}
}

type NetworkVersionRange struct {
	From network.Version
	To   network.Version
}

type NetworkHeightRange struct {
	From int64
	To   int64
}

const MaxNetworkHeight = math.MaxInt64

var NetworkVersionEpochMap = map[network.Version]NetworkHeightRange{
	network.Version0:   {From: 0, To: build.UpgradeBreezeHeight - 1},
	network.Version1:   {From: build.UpgradeBreezeHeight, To: build.UpgradeSmokeHeight},
	network.Version2:   {From: build.UpgradeSmokeHeight, To: build.UpgradeIgnitionHeight},
	network.Version3:   {From: build.UpgradeIgnitionHeight, To: build.UpgradeAssemblyHeight},
	network.Version4:   {From: build.UpgradeAssemblyHeight, To: build.UpgradeTapeHeight},
	network.Version5:   {From: build.UpgradeTapeHeight, To: build.UpgradeKumquatHeight},
	network.Version6:   {From: build.UpgradeKumquatHeight, To: build.UpgradeCalicoHeight},
	network.Version7:   {From: build.UpgradeCalicoHeight, To: build.UpgradePersianHeight},
	network.Version8:   {From: build.UpgradePersianHeight, To: build.UpgradeOrangeHeight},
	network.Version9:   {From: build.UpgradeOrangeHeight, To: build.UpgradeTrustHeight},
	network.Version10:  {From: build.UpgradeTrustHeight, To: build.UpgradeNorwegianHeight},
	network.Version11:  {From: build.UpgradeNorwegianHeight, To: build.UpgradeTurboHeight},
	network.Version12:  {From: build.UpgradeTurboHeight, To: build.UpgradeHyperdriveHeight},
	network.Version13:  {From: build.UpgradeHyperdriveHeight, To: build.UpgradeChocolateHeight},
	network.Version14:  {From: build.UpgradeChocolateHeight, To: build.UpgradeOhSnapHeight},
	network.Version15:  {From: build.UpgradeOhSnapHeight, To: build.UpgradeSkyrHeight},
	network.Version16:  {From: build.UpgradeSkyrHeight, To: int64(build.UpgradeSharkHeight)},
	network.Version17:  {From: int64(build.UpgradeSharkHeight), To: MaxNetworkHeight},
	network.VersionMax: {From: int64(build.UpgradeSharkHeight), To: MaxNetworkHeight},
}

func NetworkHeightRangeForVersion(v network.Version) (NetworkHeightRange, bool) {
	r, ok := NetworkVersionEpochMap[v]
	return r, ok
}

var (
	AllNetWorkVersions   = NetworkVersionRange{From: network.Version0, To: network.VersionMax}
	FromNetworkVersion17 = NetworkVersionRange{From: network.Version17, To: network.VersionMax}
	ToNetworkVersion16   = NetworkVersionRange{From: network.Version0, To: network.Version16}
)

var TableList = []Table{
	{
		Name:                "actor_states",
		Schema:              1,
		Task:                ActorState,
		Model:               &common.ActorState{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "actors",
		Schema:              1,
		Task:                Actor,
		Model:               &common.Actor{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "block_headers",
		Schema:              1,
		Task:                BlockHeader,
		Model:               &blocks.BlockHeader{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "block_messages",
		Schema:              1,
		Task:                BlockMessage,
		Model:               &messages.BlockMessage{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "block_parents",
		Schema:              1,
		Task:                BlockParent,
		Model:               &blocks.BlockParent{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "chain_consensus",
		Schema:              1,
		Task:                ChainConsensus,
		Model:               &chain.ChainConsensus{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "chain_economics",
		Schema:              1,
		Task:                ChainEconomics,
		Model:               &chain.ChainEconomics{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "chain_powers",
		Schema:              1,
		Task:                ChainPower,
		Model:               &power.ChainPower{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "chain_rewards",
		Schema:              1,
		Task:                ChainReward,
		Model:               &reward.ChainReward{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "data_cap_balance",
		Schema:              1,
		Task:                DataCapBalance,
		Model:               &datacap.DataCapBalance{},
		NetworkVersionRange: FromNetworkVersion17,
	},
	{
		Name:                "derived_gas_outputs",
		Schema:              1,
		Task:                GasOutputs,
		Model:               &derived.GasOutputs{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "drand_block_entries",
		Schema:              1,
		Task:                DrandBlockEntrie,
		Model:               &blocks.DrandBlockEntrie{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "id_addresses",
		Schema:              1,
		Task:                IDAddress,
		Model:               &init_.IDAddress{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "internal_messages",
		Schema:              1,
		Task:                InternalMessage,
		Model:               &messages.InternalMessage{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "internal_parsed_messages",
		Schema:              1,
		Task:                InternalParsedMessage,
		Model:               &messages.InternalParsedMessage{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "market_deal_proposals",
		Schema:              1,
		Task:                MarketDealProposal,
		Model:               &market.MarketDealProposal{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "market_deal_states",
		Schema:              1,
		Task:                MarketDealState,
		Model:               &market.MarketDealState{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "message_gas_economy",
		Schema:              1,
		Task:                MessageGasEconomy,
		Model:               &messages.MessageGasEconomy{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "messages",
		Schema:              1,
		Task:                Message,
		Model:               &messages.Message{},
		NetworkVersionRange: AllNetWorkVersions,
	},

	// added to miner_info in nv17
	{
		Name:                "miner_beneficiary",
		Schema:              1,
		Task:                MinerBeneficiary,
		Model:               &miner.MinerBeneficiary{},
		NetworkVersionRange: FromNetworkVersion17,
	},

	{
		Name:                "miner_current_deadline_infos",
		Schema:              1,
		Task:                MinerCurrentDeadlineInfo,
		Model:               &miner.MinerCurrentDeadlineInfo{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "miner_fee_debts",
		Schema:              1,
		Task:                MinerFeeDebt,
		Model:               &miner.MinerFeeDebt{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "miner_infos",
		Schema:              1,
		Task:                MinerInfo,
		Model:               &miner.MinerInfo{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "miner_locked_funds",
		Schema:              1,
		Task:                MinerLockedFund,
		Model:               &miner.MinerLockedFund{},
		NetworkVersionRange: AllNetWorkVersions,
	},

	// up to actors v8/nv16 only, ctx: https://github.com/filecoin-project/lily/issues/1076

	{
		Name:                "miner_pre_commit_infos",
		Schema:              1,
		Task:                MinerPreCommitInfo,
		Model:               &miner.MinerPreCommitInfo{},
		NetworkVersionRange: ToNetworkVersion16,
	},

	// added from nv17

	{
		Name:                "miner_pre_commit_infos",
		Schema:              1,
		Task:                MinerPreCommitInfo,
		Model:               &miner.MinerPreCommitInfoV9{},
		NetworkVersionRange: FromNetworkVersion17,
	},

	{
		Name:                "miner_sector_deals",
		Schema:              1,
		Task:                MinerSectorDeal,
		Model:               &miner.MinerSectorDeal{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "miner_sector_events",
		Schema:              1,
		Task:                MinerSectorEvent,
		Model:               &miner.MinerSectorEvent{},
		NetworkVersionRange: AllNetWorkVersions,
	},

	// added for actors v7 in network v15
	{
		Name:                "miner_sector_infos_v7",
		Schema:              1,
		Task:                MinerSectorInfoV7,
		Model:               &miner.MinerSectorInfoV7{},
		NetworkVersionRange: NetworkVersionRange{From: network.Version15, To: network.VersionMax},
	},

	// used for actors v6 and below, up to network v14
	{
		Name:                "miner_sector_infos",
		Schema:              1,
		Task:                MinerSectorInfoV1_6,
		Model:               &miner.MinerSectorInfoV1_6{},
		NetworkVersionRange: NetworkVersionRange{From: network.Version0, To: network.Version14},
	},
	{
		Name:                "miner_sector_posts",
		Schema:              1,
		Task:                MinerSectorPost,
		Model:               &miner.MinerSectorPost{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "multisig_approvals",
		Schema:              1,
		Task:                MultisigApproval,
		Model:               &msapprovals.MultisigApproval{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "multisig_transactions",
		Schema:              1,
		Task:                MultisigTransaction,
		Model:               &multisig.MultisigTransaction{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "parsed_messages",
		Schema:              1,
		Task:                ParsedMessage,
		Model:               &messages.ParsedMessage{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "power_actor_claims",
		Schema:              1,
		Task:                PowerActorClaim,
		Model:               &power.PowerActorClaim{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "receipts",
		Schema:              1,
		Task:                Receipt,
		Model:               &messages.Receipt{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "verified_registry_verifiers",
		Schema:              1,
		Task:                VerifiedRegistryVerifier,
		Model:               &verifreg.VerifiedRegistryVerifier{},
		NetworkVersionRange: AllNetWorkVersions,
	},
	{
		Name:                "verified_registry_verified_clients",
		Schema:              1,
		Task:                VerifiedRegistryVerifiedClient,
		Model:               &verifreg.VerifiedRegistryVerifiedClient{},
		NetworkVersionRange: ToNetworkVersion16,
	},
	{
		Name:                "vm_messages",
		Schema:              1,
		Task:                VMMessage,
		Model:               &messages.VMMessage{},
		NetworkVersionRange: AllNetWorkVersions,
	},
}

var (
	// TablesByName maps a table name to the table description.
	TablesByName = map[string]Table{}

	// KnownTasks is a lookup of known task names
	KnownTasks = map[string]struct{}{}

	// TablesBySchema maps a schema version to a list of tables present in that schema.
	TablesBySchema = map[int][]Table{}
)

func init() {
	for _, table := range TableList {
		TablesByName[table.Name] = table
		KnownTasks[table.Task] = struct{}{}
		TablesBySchema[table.Schema] = append(TablesBySchema[table.Schema], table)
	}
}

func TablesByTask(task string, schemaVersion int) []Table {
	tables := []Table{}
	for _, table := range TableList {
		if table.Task == task {
			tables = append(tables, table)
		}
	}
	return tables
}

func TableHeaders(v interface{}) ([]string, error) {
	q := orm.NewQuery(nil, v)
	tm := q.TableModel()
	m := tm.Table()

	if len(m.Fields) == 0 {
		return nil, fmt.Errorf("invalid table model: no fields found")
	}

	var columns []string

	for _, fld := range m.Fields {
		columns = append(columns, fld.SQLName)
	}
	return columns, nil
}

func TableSchema(v interface{}) (string, error) {
	q := orm.NewQuery(nil, v)
	tm := q.TableModel()
	m := tm.Table()

	if len(m.Fields) == 0 {
		return "", fmt.Errorf("invalid table model: no fields found")
	}

	name := strings.Trim(string(m.SQLNameForSelects), `"`)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("create table %s (\n", name))

	var fieldDefs []string
	for _, fld := range m.Fields {
		fieldDefs = append(fieldDefs, fmt.Sprintf("  %s %s", fld.Column, fld.SQLType))
	}
	sb.WriteString(strings.Join(fieldDefs, ",\n"))
	sb.WriteString("\n);\n")

	return sb.String(), nil
}

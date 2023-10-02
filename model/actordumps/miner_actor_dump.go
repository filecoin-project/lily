package actordumps

import (
	"context"
	"encoding/json"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lotus/chain/types"
)

type MinerActorDump struct {
	tableName struct{} `pg:"miner_actor_dumps"` // nolint: structcheck

	Height       int64  `pg:",pk,notnull,use_zero"`
	MinerID      string `pg:",pk,notnull"`
	MinerAddress string `pg:",pk,notnull"`
	StateRoot    string `pg:",notnull"`

	// Miner Info
	OwnerID       string `pg:",notnull"`
	OwnerAddress  string `pg:",notnull"`
	WorkerID      string `pg:",notnull"`
	WorkerAddress string `pg:",notnull"`

	ConsensusFaultedElapsed int64 `pg:",notnull,use_zero"`

	PeerID             string `pg:",notnull"`
	ControlAddresses   string `pg:",type:jsonb"`
	Beneficiary        string `pg:",notnull"`
	BeneficiaryAddress string `pg:",notnull"`

	SectorSize     uint64 `pg:",use_zero"`
	NumLiveSectors uint64 `pg:",use_zero"`

	// Claims
	RawBytePower    string `pg:"type:numeric,notnull"`
	QualityAdjPower string `pg:"type:numeric,notnull"`

	// Fil Related Fields
	// Locked Funds
	TotalLockedFunds  string `pg:"type:numeric,notnull"`
	VestingFunds      string `pg:"type:numeric,notnull"`
	InitialPledge     string `pg:"type:numeric,notnull"`
	PreCommitDeposits string `pg:"type:numeric,notnull"`

	// Balance
	AvailableBalance string `pg:"type:numeric,notnull"`
	Balance          string `pg:"type:numeric,notnull"`

	FeeDebt string `pg:"type:numeric,notnull"`
}

func (m *MinerActorDump) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerActorDump.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_actor_dumps"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

func (m *MinerActorDump) UpdateMinerInfo(minerState miner.State) error {
	minerInfo, err := minerState.Info()
	if err != nil {
		return err
	}

	m.PeerID = string(minerInfo.PeerId)
	m.WorkerID = minerInfo.Worker.String()
	m.OwnerID = minerInfo.Owner.String()
	m.ConsensusFaultedElapsed = int64(minerInfo.ConsensusFaultElapsed)
	m.SectorSize = uint64(minerInfo.SectorSize)
	m.Beneficiary = minerInfo.Beneficiary.String()

	var ctrlAddresses []string
	for _, addr := range minerInfo.ControlAddresses {
		ctrlAddresses = append(ctrlAddresses, addr.String())
	}

	b, err := json.Marshal(ctrlAddresses)
	if err == nil {
		m.ControlAddresses = string(b)
	}

	num, err := minerState.NumLiveSectors()
	if err == nil {
		m.NumLiveSectors = num
	}

	return err
}

func (m *MinerActorDump) UpdateBalanceInfo(actor *types.ActorV5, minerState miner.State) error {
	m.Balance = actor.Balance.String()

	availableBalance, err := minerState.AvailableBalance(actor.Balance)
	if err != nil {
		return err
	}
	m.AvailableBalance = availableBalance.String()

	feeDebt, err := minerState.FeeDebt()
	if err != nil {
		return err
	}
	m.FeeDebt = feeDebt.String()

	lockedFunds, err := minerState.LockedFunds()
	if err != nil {
		return err
	}
	m.InitialPledge = lockedFunds.InitialPledgeRequirement.String()
	m.VestingFunds = lockedFunds.VestingFunds.String()
	m.PreCommitDeposits = lockedFunds.PreCommitDeposits.String()
	m.TotalLockedFunds = lockedFunds.TotalLockedFunds().String()

	return nil
}

type MinerActorDumpList []*MinerActorDump

func (ml MinerActorDumpList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerActorDumpList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_actor_dumps"))

	if len(ml) == 0 {
		return nil
	}
	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}

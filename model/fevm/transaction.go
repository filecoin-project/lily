package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMTransaction struct {
	tableName struct{} `pg:"fevm_transactions"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// Hash of transaction.
	Hash string `pg:",pk,notnull"`
	// EVM network id.
	ChainID uint64 `pg:",use_zero"`
	// A sequentially incrementing counter which indicates the transaction number from the account.
	Nonce uint64 `pg:",use_zero"`
	// Hash of the block where this transaction was in.
	BlockHash string `pg:",notnull"`
	// Block number where this transaction was in.
	BlockNumber uint64 `pg:",use_zero"`
	// Integer of the transactions index position in the block.
	TransactionIndex uint64 `pg:",use_zero"`
	// ETH Address of the sender.
	From string `pg:",notnull"`
	// ETH Address of the receiver.
	To string `pg:",notnull"`
	// Amount of FIL to transfer from sender to recipient.
	Value string `pg:",notnull"`
	// Type of transactions.
	Type uint64 `pg:",use_zero"`
	// The data sent along with the transaction.
	Input string `pg:",notnull"`
	// Gas provided by the sender.
	Gas uint64 `pg:",use_zero"`
	// The maximum fee per unit of gas willing to be paid for the transaction.
	MaxFeePerGas string `pg:"type:numeric,notnull"`
	// The maximum price of the consumed gas to be included as a tip to the validator.
	MaxPriorityFeePerGas string `pg:"type:numeric,notnull"`
	AccessList           string `pg:",type:jsonb"`
	// Transaction’s signature. Recovery Identifier.
	V string `pg:",notnull"`
	// Transaction’s signature. Outputs of an ECDSA signature.
	R string `pg:",notnull"`
	// Transaction’s signature. Outputs of an ECDSA signature.
	S string `pg:",notnull"`
	// Filecoin Address of the sender.
	FromFilecoinAddress string `pg:",notnull"`
	// Filecoin Address of the receiver.
	ToFilecoinAddress string `pg:",notnull"`
	// Human-readable identifier of receiver (To).
	ToActorName string `pg:",notnull"`
	// Human-readable identifier of sender (From).
	FromActorName string `pg:",notnull"`
	// On-chain message triggering the message.
	MessageCid string `pg:",pk,notnull"`
}

func (f *FEVMTransaction) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_transactions"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMTransactionList []*FEVMTransaction

func (f FEVMTransactionList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_transactions"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}

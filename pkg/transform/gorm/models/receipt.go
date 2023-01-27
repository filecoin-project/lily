package models

import (
	types "github.com/filecoin-project/lily/pkg/transform/gorm/types"
)

type MessageReceipt struct {
	MessageCid types.DbCID `gorm:"primaryKey"`
	Receipt    Receipt     `gorm:"embedded"`

	BaseFeeBurn        types.DbToken
	OverEstimationBurn types.DbToken
	MinerPenalty       types.DbToken
	MinerTip           types.DbToken
	Refund             types.DbToken
	GasRefund          int64
	GasBurned          int64
	Error              string
}

type Receipt struct {
	Index      int64 `gorm:"primaryKey"`
	ExitCode   int64
	GasUsed    int64
	Return     []byte
	EventsRoot string
}

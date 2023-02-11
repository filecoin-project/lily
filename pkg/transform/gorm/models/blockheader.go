package models

import (
	types "github.com/filecoin-project/lily/pkg/transform/gorm/types"
)

type BlockHeaderModel struct {
	Cid           types.DbCID `gorm:"primaryKey"`
	StateRoot     types.DbCID
	Height        int64
	Miner         types.DbAddr
	ParentWeight  types.DbBigInt
	TimeStamp     uint64
	ForkSignaling uint64
	BaseFee       types.DbToken
	WinCount      int64
	ParentBaseFee types.DbToken
}

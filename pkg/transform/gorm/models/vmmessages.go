package models

import "github.com/filecoin-project/lily/pkg/transform/gorm/types"

type VmMessage struct {
	Source  types.DbCID `gorm:"primaryKey"`
	Cid     types.DbCID `gorm:"primaryKey"`
	To      types.DbAddr
	From    types.DbAddr
	Value   types.DbToken
	Method  uint64
	Params  []byte
	Receipt Receipt `gorm:"embedded"`
	Error   string
}

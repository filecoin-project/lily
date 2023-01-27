package models

import "github.com/filecoin-project/lily/pkg/transform/gorm/types"

type VmMessage struct {
	Source  types.DbCID `gorm:"primaryKey"`
	Cid     types.DbCID `gorm:"primaryKey"`
	Message Message     `gorm:"embedded"`
	Receipt Receipt     `gorm:"embedded"`
	Error   string
	Index   uint64
}

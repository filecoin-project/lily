package models

import (
	types "github.com/filecoin-project/lily/pkg/transform/gorm/types"
)

type Message struct {
	Cid        types.DbCID `gorm:"primaryKey"`
	Version    int64
	To         types.DbAddr
	From       types.DbAddr
	Nonce      uint64
	Value      types.DbToken
	GasLimit   int64
	GasFeeCap  types.DbToken
	GasPremium types.DbToken
	Method     uint64
	Params     []byte
	Signature  string `gorm:"jsonb"`
}

type ParsedMessageParams struct {
	Cid    types.DbCID `gorm:"primaryKey"`
	Params string      `gorm:"jsonb"`
	Method string
}

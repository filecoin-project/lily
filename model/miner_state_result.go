package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/ipfs/go-cid"
)

type MinerStateResult struct {
	MinerAddr address.Address
	StateRoot cid.Cid
	State     miner.State
	Info      *miner.MinerInfo
	Claim     power.Claim
}

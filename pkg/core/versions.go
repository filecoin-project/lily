package core

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

func ActorVersionForTipSet(ctx context.Context, ts *types.TipSet, ntwkVersionGetter func(ctx context.Context, epoch abi.ChainEpoch) network.Version) (actorstypes.Version, error) {
	ntwkVersion := ntwkVersionGetter(ctx, ts.Height())
	return actorstypes.VersionForNetwork(ntwkVersion)
}

var (
	AccountCodes  = cid.NewSet()
	CronCodes     = cid.NewSet()
	DataCapCodes  = cid.NewSet()
	InitCodes     = cid.NewSet()
	MarketCodes   = cid.NewSet()
	MinerCodes    = cid.NewSet()
	MultisigCodes = cid.NewSet()
	PaychCodes    = cid.NewSet()
	PowerCodes    = cid.NewSet()
	RewardCodes   = cid.NewSet()
	SystemCodes   = cid.NewSet()
	VerifregCodes = cid.NewSet()
)

func init() {
	for _, a := range []string{actors.AccountKey, actors.CronKey, actors.DatacapKey, actors.InitKey, actors.MarketKey, actors.MinerKey, actors.MultisigKey, actors.PaychKey, actors.PowerKey, actors.RewardKey, actors.SystemKey, actors.VerifregKey} {
		for _, v := range []int{0, 2, 3, 4, 5, 6, 7, 8, 9} {
			code, ok := actors.GetActorCodeID(actorstypes.Version(v), a)
			if !ok {
				continue
			}
			switch a {
			case actors.AccountKey:
				AccountCodes.Add(code)
			case actors.CronKey:
				CronCodes.Add(code)
			case actors.DatacapKey:
				DataCapCodes.Add(code)
			case actors.InitKey:
				InitCodes.Add(code)
			case actors.MarketKey:
				MarketCodes.Add(code)
			case actors.MinerKey:
				MinerCodes.Add(code)
			case actors.MultisigKey:
				MultisigCodes.Add(code)
			case actors.PaychKey:
				PaychCodes.Add(code)
			case actors.PowerKey:
				PowerCodes.Add(code)
			case actors.RewardKey:
				RewardCodes.Add(code)
			case actors.SystemKey:
				SystemCodes.Add(code)
			case actors.VerifregKey:
				VerifregCodes.Add(code)
			}
		}
	}
}

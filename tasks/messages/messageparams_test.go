package messages

import (
	"testing"

	"github.com/ipfs/go-cid"

	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	sa4builtin "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	sa5builtin "github.com/filecoin-project/specs-actors/v4/actors/builtin"

	sa0account "github.com/filecoin-project/specs-actors/actors/builtin/account"
	sa0cron "github.com/filecoin-project/specs-actors/actors/builtin/cron"
	sa0init "github.com/filecoin-project/specs-actors/actors/builtin/init"
	sa0market "github.com/filecoin-project/specs-actors/actors/builtin/market"
	sa0miner "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	sa0multisig "github.com/filecoin-project/specs-actors/actors/builtin/multisig"
	sa0paych "github.com/filecoin-project/specs-actors/actors/builtin/paych"
	sa0power "github.com/filecoin-project/specs-actors/actors/builtin/power"
	sa0reward "github.com/filecoin-project/specs-actors/actors/builtin/reward"
	sa0system "github.com/filecoin-project/specs-actors/actors/builtin/system"
	sa0verifreg "github.com/filecoin-project/specs-actors/actors/builtin/verifreg"

	sa2account "github.com/filecoin-project/specs-actors/v2/actors/builtin/account"
	sa2cron "github.com/filecoin-project/specs-actors/v2/actors/builtin/cron"
	sa2init "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	sa2market "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	sa2miner "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	sa2multisig "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
	sa2paych "github.com/filecoin-project/specs-actors/v2/actors/builtin/paych"
	sa2power "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	sa2reward "github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"
	sa2system "github.com/filecoin-project/specs-actors/v2/actors/builtin/system"
	sa2verifreg "github.com/filecoin-project/specs-actors/v2/actors/builtin/verifreg"

	sa3account "github.com/filecoin-project/specs-actors/v3/actors/builtin/account"
	sa3cron "github.com/filecoin-project/specs-actors/v3/actors/builtin/cron"
	sa3init "github.com/filecoin-project/specs-actors/v3/actors/builtin/init"
	sa3market "github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	sa3miner "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	sa3multisig "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"
	sa3paych "github.com/filecoin-project/specs-actors/v3/actors/builtin/paych"
	sa3power "github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	sa3reward "github.com/filecoin-project/specs-actors/v3/actors/builtin/reward"
	sa3system "github.com/filecoin-project/specs-actors/v3/actors/builtin/system"
	sa3verifreg "github.com/filecoin-project/specs-actors/v3/actors/builtin/verifreg"

	sa4account "github.com/filecoin-project/specs-actors/v4/actors/builtin/account"
	sa4cron "github.com/filecoin-project/specs-actors/v4/actors/builtin/cron"
	sa4init "github.com/filecoin-project/specs-actors/v4/actors/builtin/init"
	sa4market "github.com/filecoin-project/specs-actors/v4/actors/builtin/market"
	sa4miner "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	sa4multisig "github.com/filecoin-project/specs-actors/v4/actors/builtin/multisig"
	sa4paych "github.com/filecoin-project/specs-actors/v4/actors/builtin/paych"
	sa4power "github.com/filecoin-project/specs-actors/v4/actors/builtin/power"
	sa4reward "github.com/filecoin-project/specs-actors/v4/actors/builtin/reward"
	sa4system "github.com/filecoin-project/specs-actors/v4/actors/builtin/system"
	sa4verifreg "github.com/filecoin-project/specs-actors/v4/actors/builtin/verifreg"

	sa5account "github.com/filecoin-project/specs-actors/v5/actors/builtin/account"
	sa5cron "github.com/filecoin-project/specs-actors/v5/actors/builtin/cron"
	sa5init "github.com/filecoin-project/specs-actors/v5/actors/builtin/init"
	sa5market "github.com/filecoin-project/specs-actors/v5/actors/builtin/market"
	sa5miner "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	sa5multisig "github.com/filecoin-project/specs-actors/v5/actors/builtin/multisig"
	sa5paych "github.com/filecoin-project/specs-actors/v5/actors/builtin/paych"
	sa5power "github.com/filecoin-project/specs-actors/v5/actors/builtin/power"
	sa5reward "github.com/filecoin-project/specs-actors/v5/actors/builtin/reward"
	sa5system "github.com/filecoin-project/specs-actors/v5/actors/builtin/system"
	sa5verifreg "github.com/filecoin-project/specs-actors/v5/actors/builtin/verifreg"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin"
)

// TODO: generate this test
func TestMethodTableCoverage(t *testing.T) {
	type actor interface {
		Exports() []interface{}
	}
	type singleton interface {
		IsSingleton() bool
	}

	actorStates := map[cid.Cid]actor{
		sa0builtin.InitActorCodeID:             sa0init.Actor{},
		sa0builtin.MultisigActorCodeID:         sa0multisig.Actor{},
		sa0builtin.PaymentChannelActorCodeID:   sa0paych.Actor{},
		sa0builtin.RewardActorCodeID:           sa0reward.Actor{},
		sa0builtin.StorageMarketActorCodeID:    sa0market.Actor{},
		sa0builtin.StorageMinerActorCodeID:     sa0miner.Actor{},
		sa0builtin.StoragePowerActorCodeID:     sa0power.Actor{},
		sa0builtin.VerifiedRegistryActorCodeID: sa0verifreg.Actor{},
		sa0builtin.AccountActorCodeID:          sa0account.Actor{},
		sa0builtin.CronActorCodeID:             sa0cron.Actor{},
		sa0builtin.SystemActorCodeID:           sa0system.Actor{},

		// v2
		sa2builtin.InitActorCodeID:             sa2init.Actor{},
		sa2builtin.MultisigActorCodeID:         sa2multisig.Actor{},
		sa2builtin.PaymentChannelActorCodeID:   sa2paych.Actor{},
		sa2builtin.RewardActorCodeID:           sa2reward.Actor{},
		sa2builtin.StorageMarketActorCodeID:    sa2market.Actor{},
		sa2builtin.StorageMinerActorCodeID:     sa2miner.Actor{},
		sa2builtin.StoragePowerActorCodeID:     sa2power.Actor{},
		sa2builtin.VerifiedRegistryActorCodeID: sa2verifreg.Actor{},
		sa2builtin.AccountActorCodeID:          sa2account.Actor{},
		sa2builtin.CronActorCodeID:             sa2cron.Actor{},
		sa2builtin.SystemActorCodeID:           sa2system.Actor{},

		// v3
		sa3builtin.InitActorCodeID:             sa3init.Actor{},
		sa3builtin.MultisigActorCodeID:         sa3multisig.Actor{},
		sa3builtin.PaymentChannelActorCodeID:   sa3paych.Actor{},
		sa3builtin.RewardActorCodeID:           sa3reward.Actor{},
		sa3builtin.StorageMarketActorCodeID:    sa3market.Actor{},
		sa3builtin.StorageMinerActorCodeID:     sa3miner.Actor{},
		sa3builtin.StoragePowerActorCodeID:     sa3power.Actor{},
		sa3builtin.VerifiedRegistryActorCodeID: sa3verifreg.Actor{},
		sa3builtin.AccountActorCodeID:          sa3account.Actor{},
		sa3builtin.CronActorCodeID:             sa3cron.Actor{},
		sa3builtin.SystemActorCodeID:           sa3system.Actor{},

		// v4
		sa4builtin.InitActorCodeID:             sa4init.Actor{},
		sa4builtin.MultisigActorCodeID:         sa4multisig.Actor{},
		sa4builtin.PaymentChannelActorCodeID:   sa4paych.Actor{},
		sa4builtin.RewardActorCodeID:           sa4reward.Actor{},
		sa4builtin.StorageMarketActorCodeID:    sa4market.Actor{},
		sa4builtin.StorageMinerActorCodeID:     sa4miner.Actor{},
		sa4builtin.StoragePowerActorCodeID:     sa4power.Actor{},
		sa4builtin.VerifiedRegistryActorCodeID: sa4verifreg.Actor{},
		sa4builtin.AccountActorCodeID:          sa4account.Actor{},
		sa4builtin.CronActorCodeID:             sa4cron.Actor{},
		sa4builtin.SystemActorCodeID:           sa4system.Actor{},

		// v5
		sa5builtin.InitActorCodeID:             sa5init.Actor{},
		sa5builtin.MultisigActorCodeID:         sa5multisig.Actor{},
		sa5builtin.PaymentChannelActorCodeID:   sa5paych.Actor{},
		sa5builtin.RewardActorCodeID:           sa5reward.Actor{},
		sa5builtin.StorageMarketActorCodeID:    sa5market.Actor{},
		sa5builtin.StorageMinerActorCodeID:     sa5miner.Actor{},
		sa5builtin.StoragePowerActorCodeID:     sa5power.Actor{},
		sa5builtin.VerifiedRegistryActorCodeID: sa5verifreg.Actor{},
		sa5builtin.AccountActorCodeID:          sa5account.Actor{},
		sa5builtin.CronActorCodeID:             sa5cron.Actor{},
		sa5builtin.SystemActorCodeID:           sa5system.Actor{},
	}

	for code, table := range messageParamTable {
		name := builtin.ActorNameByCode(code)
		t.Run(name, func(t *testing.T) {
			if len(table) == 0 {
				t.Skipf("no method table defined for actor")
			}

			state, ok := actorStates[code]
			if !ok {
				t.Fatalf("state not found for actor code: %s", code)
			}

			exports := state.Exports()
			// Note that actor exports are 1-based
			start := 1

			if s, ok := state.(singleton); ok {
				if s.IsSingleton() {
					start = 2
				}
			}
			for i := start; i < len(exports); i++ {
				_, ok := table[int64(i)]
				if !ok {
					t.Errorf("message table missing method %d", i)
				}
			}
		})
	}
}

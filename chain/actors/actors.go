package actors

import (
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/lotus/chain/actors"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	"github.com/ipfs/go-cid"
)

type Version int

var LatestVersion = 8

var Versions = []int{0, 2, 3, 4, 5, 6, 7, 8}

const (
	Version0 Version = 0
	Version2 Version = 2
	Version3 Version = 3
	Version4 Version = 4
	Version5 Version = 5
	Version6 Version = 6
	Version7 Version = 7
	Version8 Version = 8
	Version9 Version = 9
)
const (
	AccountKey  = "account"
	CronKey     = "cron"
	InitKey     = "init"
	MarketKey   = "storagemarket"
	MinerKey    = "storageminer"
	MultisigKey = "multisig"
	PaychKey    = "paymentchannel"
	PowerKey    = "storagepower"
	RewardKey   = "reward"
	SystemKey   = "system"
	VerifregKey = "verifiedregistry"
	DatacapKey  = "datacap"
)

// GetActorCodeID looks up a builtin actor's code CID by actor version and canonical actor name.
func GetActorCodeID(av Version, name string) (cid.Cid, bool) {
	// Actors V8 and above
	if c, ok := actors.GetActorCodeID(actorstypes.Version(av), name); ok {
		return c, true
	}

	// Actors V7 and lower
	switch name {

	case AccountKey:
		switch av {

		case Version0:
			return builtin0.AccountActorCodeID, true

		case Version2:
			return builtin2.AccountActorCodeID, true

		case Version3:
			return builtin3.AccountActorCodeID, true

		case Version4:
			return builtin4.AccountActorCodeID, true

		case Version5:
			return builtin5.AccountActorCodeID, true

		case Version6:
			return builtin6.AccountActorCodeID, true

		case Version7:
			return builtin7.AccountActorCodeID, true
		}

	case CronKey:
		switch av {

		case Version0:
			return builtin0.CronActorCodeID, true

		case Version2:
			return builtin2.CronActorCodeID, true

		case Version3:
			return builtin3.CronActorCodeID, true

		case Version4:
			return builtin4.CronActorCodeID, true

		case Version5:
			return builtin5.CronActorCodeID, true

		case Version6:
			return builtin6.CronActorCodeID, true

		case Version7:
			return builtin7.CronActorCodeID, true
		}

	case InitKey:
		switch av {

		case Version0:
			return builtin0.InitActorCodeID, true

		case Version2:
			return builtin2.InitActorCodeID, true

		case Version3:
			return builtin3.InitActorCodeID, true

		case Version4:
			return builtin4.InitActorCodeID, true

		case Version5:
			return builtin5.InitActorCodeID, true

		case Version6:
			return builtin6.InitActorCodeID, true

		case Version7:
			return builtin7.InitActorCodeID, true
		}

	case MarketKey:
		switch av {

		case Version0:
			return builtin0.StorageMarketActorCodeID, true

		case Version2:
			return builtin2.StorageMarketActorCodeID, true

		case Version3:
			return builtin3.StorageMarketActorCodeID, true

		case Version4:
			return builtin4.StorageMarketActorCodeID, true

		case Version5:
			return builtin5.StorageMarketActorCodeID, true

		case Version6:
			return builtin6.StorageMarketActorCodeID, true

		case Version7:
			return builtin7.StorageMarketActorCodeID, true
		}

	case MinerKey:
		switch av {

		case Version0:
			return builtin0.StorageMinerActorCodeID, true

		case Version2:
			return builtin2.StorageMinerActorCodeID, true

		case Version3:
			return builtin3.StorageMinerActorCodeID, true

		case Version4:
			return builtin4.StorageMinerActorCodeID, true

		case Version5:
			return builtin5.StorageMinerActorCodeID, true

		case Version6:
			return builtin6.StorageMinerActorCodeID, true

		case Version7:
			return builtin7.StorageMinerActorCodeID, true
		}

	case MultisigKey:
		switch av {

		case Version0:
			return builtin0.MultisigActorCodeID, true

		case Version2:
			return builtin2.MultisigActorCodeID, true

		case Version3:
			return builtin3.MultisigActorCodeID, true

		case Version4:
			return builtin4.MultisigActorCodeID, true

		case Version5:
			return builtin5.MultisigActorCodeID, true

		case Version6:
			return builtin6.MultisigActorCodeID, true

		case Version7:
			return builtin7.MultisigActorCodeID, true
		}

	case PaychKey:
		switch av {

		case Version0:
			return builtin0.PaymentChannelActorCodeID, true

		case Version2:
			return builtin2.PaymentChannelActorCodeID, true

		case Version3:
			return builtin3.PaymentChannelActorCodeID, true

		case Version4:
			return builtin4.PaymentChannelActorCodeID, true

		case Version5:
			return builtin5.PaymentChannelActorCodeID, true

		case Version6:
			return builtin6.PaymentChannelActorCodeID, true

		case Version7:
			return builtin7.PaymentChannelActorCodeID, true
		}

	case PowerKey:
		switch av {

		case Version0:
			return builtin0.StoragePowerActorCodeID, true

		case Version2:
			return builtin2.StoragePowerActorCodeID, true

		case Version3:
			return builtin3.StoragePowerActorCodeID, true

		case Version4:
			return builtin4.StoragePowerActorCodeID, true

		case Version5:
			return builtin5.StoragePowerActorCodeID, true

		case Version6:
			return builtin6.StoragePowerActorCodeID, true

		case Version7:
			return builtin7.StoragePowerActorCodeID, true
		}

	case RewardKey:
		switch av {

		case Version0:
			return builtin0.RewardActorCodeID, true

		case Version2:
			return builtin2.RewardActorCodeID, true

		case Version3:
			return builtin3.RewardActorCodeID, true

		case Version4:
			return builtin4.RewardActorCodeID, true

		case Version5:
			return builtin5.RewardActorCodeID, true

		case Version6:
			return builtin6.RewardActorCodeID, true

		case Version7:
			return builtin7.RewardActorCodeID, true
		}

	case SystemKey:
		switch av {

		case Version0:
			return builtin0.SystemActorCodeID, true

		case Version2:
			return builtin2.SystemActorCodeID, true

		case Version3:
			return builtin3.SystemActorCodeID, true

		case Version4:
			return builtin4.SystemActorCodeID, true

		case Version5:
			return builtin5.SystemActorCodeID, true

		case Version6:
			return builtin6.SystemActorCodeID, true

		case Version7:
			return builtin7.SystemActorCodeID, true
		}

	case VerifregKey:
		switch av {

		case Version0:
			return builtin0.VerifiedRegistryActorCodeID, true

		case Version2:
			return builtin2.VerifiedRegistryActorCodeID, true

		case Version3:
			return builtin3.VerifiedRegistryActorCodeID, true

		case Version4:
			return builtin4.VerifiedRegistryActorCodeID, true

		case Version5:
			return builtin5.VerifiedRegistryActorCodeID, true

		case Version6:
			return builtin6.VerifiedRegistryActorCodeID, true

		case Version7:
			return builtin7.VerifiedRegistryActorCodeID, true
		}
	}

	return cid.Undef, false
}

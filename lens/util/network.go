package util

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/stmgr"
)

// DefaultNetwork is the filecoing network this version of visor has been built against
var DefaultNetwork = NewNetwork(stmgr.DefaultUpgradeSchedule(), build.NewestNetworkVersion)

// Network holds properties of the filecoin network
type Network struct {
	networkVersions []versionSpec
	latestVersion   network.Version
}

type versionSpec struct {
	networkVersion network.Version
	atOrBelow      abi.ChainEpoch
}

func NewNetwork(us stmgr.UpgradeSchedule, current network.Version) *Network {
	var networkVersions []versionSpec
	lastVersion := network.Version0
	if len(us) > 0 {
		for _, upgrade := range us {
			networkVersions = append(networkVersions, versionSpec{
				networkVersion: lastVersion,
				atOrBelow:      upgrade.Height,
			})
			lastVersion = upgrade.Network
		}
	} else {
		lastVersion = current
	}

	return &Network{
		networkVersions: networkVersions,
		latestVersion:   lastVersion,
	}
}

func (n *Network) Version(ctx context.Context, height abi.ChainEpoch) network.Version {
	// The epochs here are the _last_ epoch for every version, or -1 if the
	// version is disabled.
	for _, spec := range n.networkVersions {
		if height <= spec.atOrBelow {
			return spec.networkVersion
		}
	}
	return n.latestVersion
}

package commands

import (
	"github.com/urfave/cli/v2"
)

var clientAPIFlags struct {
	apiAddr  string
	apiToken string
}

var clientAPIFlag = &cli.StringFlag{
	Name:        "api",
	Usage:       "Address of visor api in multiaddr format.",
	EnvVars:     []string{"VISOR_API"},
	Value:       "/ip4/127.0.0.1/tcp/1234",
	Destination: &clientAPIFlags.apiAddr,
}

var clientTokenFlag = &cli.StringFlag{
	Name:        "api-token",
	Usage:       "Authentication token for visor api.",
	EnvVars:     []string{"VISOR_API_TOKEN"},
	Value:       "",
	Destination: &clientAPIFlags.apiToken,
}

// clientAPIFlagSet are used by commands that act as clients of a daemon's API
var clientAPIFlagSet = []cli.Flag{
	clientAPIFlag,
	clientTokenFlag,
}

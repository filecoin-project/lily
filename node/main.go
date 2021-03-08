package main

import (
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/node/repo"

	lotuscli "github.com/filecoin-project/lotus/cli"

	cli2 "github.com/filecoin-project/sentinel-visor/node/cli"
)

func main() {
	build.RunningNodeType = build.NodeFull
	lotuslog.SetupLogLevels()
	app := &cli.App{
		Name:                 "lily",
		Usage:                "filecoin blockchain monitoring and analysis",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"SENTINEL_LOTUS_PATH"},
				Hidden:  true,
				Value:   "~/.lotus", // TODO: Consider XDG_DATA_HOME
			},
		},
		Commands: []*cli.Command{
			cli2.DaemonCmd,
			cli2.SentinelStartWatchCmd,
			lotuscli.NetCmd,
			lotuscli.StateCmd,
			lotuscli.SyncCmd,
		},
	}
	app.Setup()
	app.Metadata["repoType"] = repo.FullNode
	lcli.RunApp(app)
}

package commands

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
)

var HelpModelsListCmd = &cli.Command{
	Name: "models-list",
	Action: func(cctx *cli.Context) error { // initialize tabwriter
		t := table.NewWriter()
		t.AppendHeader(table.Row{"Model", "Description"})
		for _, m := range tasktype.AllTableTasks {
			comment := tasktype.TableComment[m]
			t.AppendRow(table.Row{m, comment})
			t.AppendSeparator()
		}
		fmt.Println(t.Render())
		return nil
	},
}

var HelpModelsDescribeCmd = &cli.Command{
	Name: "models-describe",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().First() == "" {
			return xerrors.Errorf("model name required, run `lily help models-list`, to see all available models")
		}
		mname := cctx.Args().First()
		if _, found := tasktype.TableLookup[mname]; !found {
			return xerrors.Errorf("model %s doesn't exist", mname)
		}

		modelFields := tasktype.TableFieldComments[mname]
		t := table.NewWriter()
		t.AppendHeader(table.Row{"Fields", "Description"})
		t.SortBy([]table.SortBy{
			{Name: "Fields", Mode: table.Asc}})
		t.SetCaption(tasktype.TableComment[mname])
		for field, comment := range modelFields {
			t.AppendRow(table.Row{field, comment})
			t.AppendSeparator()
		}
		fmt.Println(t.Render())
		return nil
	},
}

var HelpCmd = &cli.Command{
	Name:      "help",
	Aliases:   []string{"h"},
	Usage:     "Shows a list of commands or help for one command",
	ArgsUsage: "[command]",
	Subcommands: []*cli.Command{
		HelpModelsListCmd,
		HelpModelsDescribeCmd,
	},
}

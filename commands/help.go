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
	Action: func(c *cli.Context) error {
		args := c.Args()
		if args.Present() {
			return ShowCommandHelp(c, args.First())
		}

		_ = cli.ShowAppHelp(c)
		return nil
	},
}

func ShowCommandHelp(ctx *cli.Context, command string) error {
	if command == "" {
		cli.HelpPrinter(ctx.App.Writer, cli.SubcommandHelpTemplate, ctx.App)
		return nil
	}

	for _, c := range ctx.App.Commands {
		if c.HasName(command) {
			templ := c.CustomHelpTemplate
			if templ == "" {
				templ = cli.CommandHelpTemplate
			}

			cli.HelpPrinter(ctx.App.Writer, templ, c)

			return nil
		}
	}

	for _, t := range helpTopics {
		if t.Name == command {
			fmt.Fprintln(ctx.App.Writer, t.Text)
			return nil
		}
	}

	if ctx.App.CommandNotFound == nil {
		return cli.Exit(fmt.Sprintf("No help topic for '%v'", command), 3)
	}

	ctx.App.CommandNotFound(ctx, command)
	return nil
}

func Metadata() map[string]interface{} {
	return map[string]interface{}{
		"Topics": helpTopics,
	}
}

var AppHelpTemplate = `{{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

Usage:

  {{.HelpName}} [global options] <command> [arguments...]

The commands are:
{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

Use "{{.HelpName}} help <command>" for more information about a command.

Additional help topics:
{{range .Metadata.Topics}}
  {{.Name}}{{"\t"}}{{.Description}}{{end}}

Use "{{.HelpName}} help <topic>" for more information about that topic.
`

type helpTopic struct {
	Name        string
	Description string
	Text        string
}

// ----------------------------------------------------------------------------
//                                                            80 characters -->
var helpTopics = []helpTopic{
	{
		Name:        "overview",
		Description: "Overview of visor",
		Text: `Visor is an application for capturing on-chain state from the filecoin network.
It extracts data from the blocks and messages contained in each tipset and
captures the effects those messages have on actor states. Visor can 'watch'
the head of the filecoin chain for incoming tipsets or 'walk' the chain to
analyze historic tipsets.

A watch is intended to follow the growth of the chain and operates by
subscribing to incoming tipsets and processing them as they arrive. A
confidence level may be  specified which determines how many epochs visor
should wait before processing the tipset. This is to allow for chain
reorganisation near the head. A low confidence level risks extracting data from
tipsets that do not form part of the consensus chain.

A walk takes a range of heights and will walk from the heaviest tipset at the
upper height to the lower height using the parent state root present in each
tipset.

The type of data extracted by lily is controlled by 'tasks' that focus on
particular parts of the chain. For more information about available tasks
see 'visor help tasks'.

Data is extracted into models that represent chain objects, components of actor
state and derived statistics. Visor can insert these extracted models into a
TimescaleDB database as separate tables or emit them as csv files.

Visor requires access to a filecoin blockstore that holds the state of the
chain. For watching incoming tipsets the blockstore must be connected and in
sync with the filecoin network. Historic walks can be performed against an
offline store.

While running, lily will maintain its own local blockstore and attempt
to synchronise it with the filecoin network. For more information on running
visor as a daemon, including how to initialise the blockstore, see
'visor help daemon'.

`,
	},

	{
		Name:        "monitoring",
		Description: "Monitoring lily operation",
		Text: `Visor may be monitored during operation using logfiles, metrics and tracing.
The lily command recognizes environment variables and provides options to
control the behaviour of each type of monitoring output. Options should be
supplied before any sub command:

  lily [global options] <command>

Visor uses the IPFS logging library (https://github.com/ipfs/go-ipfs) to write
application logs. By default logs are written to STDERR in JSON format. Log
lines are labeled with one of seven levels to indicate severity of the message
(DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL). Line are also labeled with
named systems which indicate the area of function that produced the log. Each
system may be configured to only emit log messages of a specific level or
higher. By default all log levels are set to debug and above.

A number of environment variables may be used to control the format and
destination of the logs.

  GOLOG_LOG_LEVEL        Set the default log level for all log systems.

  GOLOG_LOG_FMT          Set the output log format. By default logs will be
                         colorized and text format. Use 'json' to specify
                         JSON formatted logs and 'nocolor' to log in text
                         format without colors.

  GOLOG_FILE             Specify the name of the file that logs should be
                         written to. Only used if GOLOG_OUTPUT contains the
                         'file' keyword.

  GOLOG_OUTPUT           Specify whether to output to file, stderr, stdout or
                         a combination. Separate each keyword with a '+', for
                         example: file+stderr

  LILY_LOG_LEVEL_NAMED  Set the log level of specific loggers. The value
                         should be a comma delimited list of log systems and
                         log levels formatted as name:level, for example
                         'logger1:debug,logger2:info'.

In addition, lily supports some global options for controlling logging:

  --log-level LEVEL        Set the default log level for all loggers to LEVEL.
                           This option overrides any value set using the
                           GOLOG_LOG_LEVEL environment variable.

  --log-level-named value  Set the log level of specific loggers. This option
                           overrides any value set using the
                           LILY_LOG_LEVEL_NAMED environment variable.

To control logging output while the lily daemon is running see 'visor help log'.

During operation lily exposes metrics and debugging information on port 9991
by default. The address used by this http server can be changed using the
'--prometheus-port' option which expects an IP address and port number. The
address may be omitted to run the server on all interfaces, for example: ':9991'.

The following paths can be accessed using a standard web browser.

  /metrics       Metrics published in prometheus format

  /debug/pprof/  Access to standard Go profiling and debugging information
                 memory allocations, cpu profile and active goroutines dumps.

Visor can publish function level tracing to a Jaeger compatible service. By
default tracing is disabled.

Environment variables for controlling function level tracing:

  LILY_TRACING         Enable tracing. Set to 'true' to enable tracing.

  JAEGER_AGENT_HOST,    Hostname and port of a Jaeger compatible agent that
  JAEGER_AGENT_PORT     lily should send traces to.

  JAEGER_SERVICE_NAME   The name lily should use when reporting traces.

  JAEGER_SAMPLER_TYPE,  Control the type of sampling used to capture traces.
  JAEGER_SAMPLER_PARAM  The type may be either 'const' or 'probabilistic'.
                        The behaviour of the sampler is controlled by the
                        value of param. For a 'const' sampler a value of 1
                        indicates that every function call should be traced,
                        while 0 means none should be traced. No intermediate
                        values are accepted. For a 'probabilistic' sampler
                        the param indicates the fraction of function calls
                        that should be sampled.

The following options may be used to override the tracing environment variables:

  --tracing
  --jaeger-agent-host
  --jaeger-agent-port
  --jaeger-service-name
  --jaeger-sampler-type
  --jaeger-sampler-param
`,
	},
}

package job

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"

	lotuscli "github.com/filecoin-project/lotus/cli"
)

//revive:disable
var GapFindCmd = &cli.Command{
	Name:  "find",
	Usage: "find gaps in the database for a given range and a set of tasks.",
	Description: `
The find job searches for gaps in a database storage system by executing the SQL gap_find() function over the visor_processing_reports table.
find will query the database for gaps based on the list of tasks (--tasks) provided over the specified range (--from --to).
An epoch is considered to have gaps iff:
- a task specified by the --task flag is not present at each epoch within the specified range.
- a task specified by the --task flag does not have status 'OK' at each epoch within the specified range.
The results of the find job are written to the visor_gap_reports table with status 'GAP'.

As an example, the below command:
 $ lily job run --tasks=block_header,messages find --from=10 --to=20
searches for gaps in block_header and messages tasks from epoch 10 to 20 (inclusive). 

Constraints:
- the find job must NOT be executed against heights that were imported from historical data dumps: https://lilium.sh/data/dumps/ 
since visor_processing_report entries will not be present for imported data (meaning the entire range will be considered to have gaps).
- the find job must be executed BEFORE a fill job. These jobs must NOT be executed simultaneously.
`,
	Flags: []cli.Flag{
		RangeFromFlag,
		RangeToFlag,
	},
	Before: func(_ *cli.Context) error {
		tasks := RunFlags.Tasks.Value()
		for _, taskName := range tasks {
			if _, found := tasktype.TaskLookup[taskName]; found {
				continue
			} else if _, found := tasktype.TableLookup[taskName]; found {
				continue
			} else {
				return fmt.Errorf("unknown task: %s", taskName)
			}
		}
		return rangeFlags.validate()
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyGapFind(ctx, &lily.LilyGapFindConfig{
			JobConfig: RunFlags.ParseJobConfig("find"),
			To:        rangeFlags.to,
			From:      rangeFlags.from,
		})
		if err != nil {
			return err
		}
		return commands.PrintNewJob(os.Stdout, res)
	},
}

package sqlrepo

import (
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/sqlrepo/tstracker"
	"github.com/filecoin-project/sentinel-visor/lens/util"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	size := c.Int("lens-cache-hint")
	// don't set cache size when it's the default.
	if size == 1024*1024 {
		size = -1
	}
	doLog := false
	if strings.Contains(c.String("log-level-named"), "postgres") {
		doLog = true
	}
	bs, err := tstracker.NewTrackingPgChainstore(c.Context, c.String("repo"), c.String("lens-postgres-namespace"), doLog, size)
	if err != nil {
		return nil, nil, err
	}

	return util.NewAPIOpener(c.Context, bs, bs.GetCurrentTipset, 0)
}

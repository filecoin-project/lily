package sqlrepo

import (
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/sqlrepo/tstracker"
	"github.com/filecoin-project/sentinel-visor/lens/util"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	bs, err := tstracker.NewTrackingPgChainstore(c.Context, c.String("repo"))
	if err != nil {
		return nil, nil, err
	}

	return util.NewAPIOpener(c.Context, bs, bs.GetCurrentTipset, c.Int("lens-cache-hint"))
}

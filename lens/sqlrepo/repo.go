package sqlrepo

import (
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
	"github.com/urfave/cli/v2"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	bs, err := NewBlockStore(c.String("repo"))
	if err != nil {
		return nil, nil, err
	}

	return util.NewAPIOpener(c, bs, bs.(*SqlBlockstore).getMasterTsKey)
}

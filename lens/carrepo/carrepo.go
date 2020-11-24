package carrepo

import (
	"context"
	"fmt"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/ulimit"
	"github.com/filecoin-project/sentinel-visor/lens/util"
	"github.com/willscott/carbs"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	if _, _, err := ulimit.ManageFdLimit(); err != nil {
		return nil, nil, fmt.Errorf("setting file descriptor limit: %s", err)
	}

	db, err := carbs.Load(c.String("repo"), false)
	if err != nil {
		return nil, nil, err
	}
	cacheDB := util.NewCachingStore(db)

	h := func(ctx context.Context, lookback int) (*types.TipSetKey, error) {
		c, err := db.Roots()
		if err != nil {
			return nil, err
		}
		tsk := types.NewTipSetKey(c...)
		return &tsk, nil
	}
	return util.NewAPIOpener(c, cacheDB, h)
}

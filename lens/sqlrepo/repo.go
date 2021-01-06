package sqlrepo

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/sqlrepo/annotated"
	"github.com/filecoin-project/sentinel-visor/lens/util"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	store, err := annotated.NewPgChainStore(c.Context, c.String("repo"))
	if err != nil {
		return nil, nil, err
	}

	headMthd := func(ctx context.Context, _ int) (*types.TipSetKey, error) {
		cids := store.GetCurrentTipset(ctx)
		tsk := types.NewTipSetKey(cids...)
		return &tsk, nil
	}

	return util.NewAPIOpener(c, store, headMthd)
}

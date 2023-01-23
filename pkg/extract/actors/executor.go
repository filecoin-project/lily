package actors

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/tasks"
)

func ExecuteStateDiff(ctx context.Context, grp *errgroup.Group, api tasks.DataSource, act *ActorChange, fns ...ActorStateDiff) ([]ActorStateChange, error) {
	out := make([]ActorStateChange, len(fns))
	for i, fn := range fns {
		fn := fn
		i := i

		grp.Go(func() error {
			res, err := fn.Diff(ctx, api, act)
			if err != nil {
				return err
			}
			out[i] = res
			return nil
		})
	}

	if err := grp.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

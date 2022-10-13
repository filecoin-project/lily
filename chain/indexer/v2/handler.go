package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/cborable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	cborable2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/cborable"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("indexmanager")

type Manager struct {
	indexer *TipSetIndexer
	stuff   *ThingIDK
	api     tasks.DataSource
	strg    model.Storage
}

func NewIndexManager(strg model.Storage, api tasks.DataSource, tasks []string) (*Manager, error) {
	stuff, err := GetTransformersForTasks(tasks...)
	if err != nil {
		return nil, err
	}
	return &Manager{
		indexer: NewTipSetIndexer(api, stuff.Tasks, 1024),
		stuff:   stuff,
		api:     api,
		strg:    strg,
	}, nil
}

func (m *Manager) TipSet(ctx context.Context, ts *types.TipSet, options ...indexer.Option) (bool, error) {
	parent, err := m.api.TipSet(ctx, ts.Parents())
	if err != nil {
		return false, err
	}
	transformer, consumer, err := m.startRouters(ctx,
		[]transform.Handler{cborable2.NewCborTransform()},
		[]load.Handler{cborable.NewCarResultConsumer(ts, parent)},
	)
	if err != nil {
		return false, err
	}

	start := time.Now()
	results, err := m.indexer.TipSet(ctx, ts)
	if err != nil {
		return false, err
	}

	grp := errgroup.Group{}
	grp.Go(func() (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("transform routine receieved error: %w", err)
			}
			stopErr := transformer.Stop()
			if stopErr != nil {
				err = fmt.Errorf("%s stopping transfrom: %w", err, stopErr)
			}
		}()
		for res := range results {
			if len(res.State().Data) > 0 {
				if err := transformer.Route(ctx, res); err != nil {
					return err
				}
			}
		}
		return
	})
	grp.Go(func() (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("consumer routine receieved error: %w", err)
			}
			stopErr := consumer.Stop()
			if stopErr != nil {
				err = fmt.Errorf("%s stopping consumer: %w", err, stopErr)
			}
		}()
		for res := range transformer.Results() {
			if err := consumer.Route(ctx, res); err != nil {
				return err
			}
		}
		return
	})
	if err := grp.Wait(); err != nil {
		return false, err
	}
	log.Infow("index complete", "duration", time.Since(start))
	return true, nil
}

type Transformer interface {
	Route(ctx context.Context, data transform.IndexState) error
	Results() chan transform.Result
	Stop() error
}

type Loader interface {
	Route(ctx context.Context, data transform.Result) error
	Stop() error
}

func (m *Manager) startRouters(ctx context.Context, handlers []transform.Handler, consumers []load.Handler) (Transformer, Loader, error) {
	tr, err := transform.NewRouter(m.stuff.Tasks, handlers...)
	if err != nil {
		return nil, nil, err
	}
	tr.Start(ctx)

	lr, err := load.NewRouter(consumers...)
	if err != nil {
		return nil, nil, err
	}
	lr.Start(ctx)

	return tr, lr, nil
}

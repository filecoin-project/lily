package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/cborable"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	cborable2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/cborable"
	persistable2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/tasks"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("indexmanager")

type Manager struct {
	extractor    *extract.StateExtractor
	tsTransforms *persistable2.TipSetTaskTransforms
	asTransforms *persistable2.ActorTaskTransforms
	api          tasks.DataSource
	strg         model.Storage
	reporter     string
}

func NewIndexManager(strg model.Storage, api tasks.DataSource, tasks []string, reporter string) (*Manager, error) {
	tsTransforms, asTransforms, modelTasks, err := persistable2.GetTransformersForTasks(tasks...)
	if err != nil {
		return nil, err
	}

	extractor, err := extract.NewStateExtractor(api, modelTasks, 1024, 1024, 1024)
	if err != nil {
		return nil, err
	}
	return &Manager{
		extractor:    extractor,
		tsTransforms: tsTransforms,
		asTransforms: asTransforms,
		api:          api,
		strg:         strg,
		reporter:     reporter,
	}, nil
}

func (m *Manager) TipSet(ctx context.Context, ts *types.TipSet, options ...indexer.Option) (ok bool, err error) {
	start := time.Now()
	// in case something shits the bed
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Errorf("indexer recovered from panic %v", r)
			log.Errorf("%s", errMsg)
			ok = false
			err = errMsg
		}
	}()

	parent, err := m.api.TipSet(ctx, ts.Parents())
	if err != nil {
		return false, err
	}

	tsTrns, actTrns, consumer, err := m.startAllRouters(ctx, m.reporter,
		append(m.tsTransforms.Transformers, cborable2.NewCborTipSetTransform()),
		append(m.asTransforms.Transformers, cborable2.NewCborActorTransform()),
		[]load.Handler{
			&persistable.PersistableResultConsumer{Strg: m.strg},
			cborable.NewCarResultConsumer(ts, parent),
		},
	)
	if err != nil {
		return false, err
	}

	tsResults, actResults, err := m.extractor.Start(ctx, ts, parent)
	if err != nil {
		return false, err
	}

	success := atomic.NewBool(true)
	grp := errgroup.Group{}
	grp.Go(func() (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("tipset transform routine receieved error: %w", err)
			}
			stopErr := tsTrns.Stop()
			if stopErr != nil {
				err = fmt.Errorf("%s stopping tipset transfrom: %w", err, stopErr)
			}
		}()

		for res := range tsResults {
			// if any result failed to complete we did not index this tipset successfully.
			if res.Error != nil {
				log.Warnw("failed to complete task", "name", res.Error)
				success.Store(false)
			}
			if err := tsTrns.Route(ctx, res); err != nil {
				return err
			}
		}
		return
	})
	grp.Go(func() (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("actor transform routine receieved error: %w", err)
			}
			stopErr := actTrns.Stop()
			if stopErr != nil {
				err = fmt.Errorf("%s stopping actor transfrom: %w", err, stopErr)
			}
		}()

		for res := range actResults {
			// if any result failed to complete we did not index this tipset successfully.
			if len(res.Results.Errors()) > 0 {
				log.Warnw("failed to complete task", "name", res.Results.Errors())
				success.Store(false)
			}
			if err := actTrns.Route(ctx, res); err != nil {
				return err
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

		subGrp := errgroup.Group{}

		subGrp.Go(func() error {
			for res := range tsTrns.Results() {
				if err := consumer.Route(ctx, res); err != nil {
					return err
				}
			}
			return nil
		})
		subGrp.Go(func() error {
			for res := range actTrns.Results() {
				if err := consumer.Route(ctx, res); err != nil {
					return err
				}
			}
			return nil
		})
		return subGrp.Wait()
	})
	if err := grp.Wait(); err != nil {
		return false, err
	}
	log.Infow("stopping indexer", "duration", time.Since(start), "success", success.Load())
	return success.Load(), nil
}

type ActorTransformer interface {
	Route(ctx context.Context, data *extract.ActorStateResult) error
	Results() chan transform.Result
	Stop() error
}

type TipSetTransformer interface {
	Route(ctx context.Context, data *extract.TipSetStateResult) error
	Results() chan transform.Result
	Stop() error
}

type Loader interface {
	Route(ctx context.Context, data transform.Result) error
	Stop() error
}

func (m *Manager) startAllRouters(ctx context.Context, reporter string, tsHandlers []transform.TipSetStateHandler, actHandlers []transform.ActorStateHandler, consumers []load.Handler) (TipSetTransformer, ActorTransformer, Loader, error) {
	tsr, err := transform.NewTipSetStateRouter(reporter, tsHandlers...)
	if err != nil {
		return nil, nil, nil, err
	}
	tsr.Start(ctx)

	asr, err := transform.NewActorStateRouter(reporter, actHandlers...)
	if err != nil {
		return nil, nil, nil, err
	}
	asr.Start(ctx)

	lr, err := load.NewRouter(consumers...)
	if err != nil {
		return nil, nil, nil, err
	}
	lr.Start(ctx)

	return tsr, asr, lr, nil
}

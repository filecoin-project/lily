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
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/system"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("indexmanager")

type Manager struct {
	indexer    *TipSetIndexer
	transforms *TaskTransforms
	api        tasks.DataSource
	strg       model.Storage
}

func NewIndexManager(strg model.Storage, api tasks.DataSource, tasks []string) (*Manager, error) {
	transforms, err := GetTransformersForTasks(tasks...)
	if err != nil {
		return nil, err
	}
	idxer, err := NewTipSetIndexer(api, transforms.Tasks, 1024)
	if err != nil {
		return nil, err
	}
	return &Manager{
		indexer:    idxer,
		transforms: transforms,
		api:        api,
		strg:       strg,
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

	transformer, consumer, err := m.startRouters(ctx,
		append(m.transforms.Transformers, system.NewProcessingReportTransform()),
		[]load.Handler{&persistable.PersistableResultConsumer{Strg: m.strg, GetName: GetLegacyTaskNameForTransform()}},
	)
	if err != nil {
		return false, err
	}

	parent, err := m.api.TipSet(ctx, ts.Parents())
	if err != nil {
		return false, err
	}

	results, err := m.indexer.TipSet(ctx, ts, parent)
	if err != nil {
		return false, err
	}

	success := atomic.NewBool(true)
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
			// if any result failed to complete we did not index this tipset successfully.
			if !res.Complete() {
				log.Warnw("failed to complete task", "name", res.Task().String())
				success.Store(false)
			}
			if len(res.Models()) > 0 {
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
	log.Infow("stopping indexer", "duration", time.Since(start), "success", success.Load())
	return success.Load(), nil
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
	tr, err := transform.NewRouter(m.transforms.Tasks, handlers...)
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

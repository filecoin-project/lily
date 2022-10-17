package v2

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/cborable"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	tasks2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/tasks"
	"github.com/filecoin-project/lily/model"
	v2 "github.com/filecoin-project/lily/model/v2"
)

const BitWidth = 8

type Feeder struct {
	Strg model.Storage
}

func (f *Feeder) Index(ctx context.Context, path string) error {
	fi, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fi.Close()
	mr, err := cborable.NewModelReader(ctx, fi)
	if err != nil {
		return err
	}

	for _, state := range mr.States() {
		tipset := state.Current

		tasks, err := mr.ModelMetasForTipSet(tipset.Key())
		if err != nil {
			return err
		}

		ts, as, err := tasks2.GetTransformersForModelMeta(tasks)
		if err != nil {
			return err
		}
		tsTransforms, asTransforms, consumer, err := f.startAllRouters(ctx, "feeder", ts, as, []load.Handler{&persistable.PersistableResultConsumer{Strg: f.Strg}})
		if err != nil {
			return err
		}

		// TODO make this a method, or shadow all the above vars
		grp := errgroup.Group{}
		grp.Go(func() error {
			defer func() {
				if err := tsTransforms.Stop(); err != nil {
					log.Errorw("stopping tipset transformer", "error", err)
				}
				if err := asTransforms.Stop(); err != nil {
					log.Errorw("stopping actor transformer", "error", err)
				}
			}()
			for _, task := range tasks {
				data, err := mr.GetModels(tipset.Key(), task)
				if err != nil {
					return err
				}
				switch task.Kind {
				case v2.ModelActorKind:
					err := asTransforms.Route(ctx, &extract.ActorStateResult{
						Task:      task,
						TipSet:    tipset,
						Results:   &ActorResultsImpl{models: data},
						StartTime: time.Now(),
						Duration:  0,
					})
					if err != nil {
						return err
					}
				case v2.ModelTsKind:
					err := tsTransforms.Route(ctx, &extract.TipSetStateResult{
						Task:      task,
						TipSet:    tipset,
						StartTime: time.Now(),
						Duration:  0,
						Models:    data,
						Error:     nil,
					})
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("unknown task kind %s", task.Kind)
				}
			}
			return nil
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
				for res := range tsTransforms.Results() {
					if err := consumer.Route(ctx, res); err != nil {
						return err
					}
				}
				return nil
			})
			subGrp.Go(func() error {
				for res := range asTransforms.Results() {
					if err := consumer.Route(ctx, res); err != nil {
						return err
					}
				}
				return nil
			})
			return subGrp.Wait()
		})
		if err := grp.Wait(); err != nil {
			return err
		}

	}

	return nil
}

type ActorResultsImpl struct {
	models []v2.LilyModel
}

func (a *ActorResultsImpl) Models() []v2.LilyModel {
	return a.models
}

func (a *ActorResultsImpl) Errors() []error {
	return nil
}

func (f *Feeder) startAllRouters(ctx context.Context, reporter string, tsHandlers []transform.TipSetStateHandler, actHandlers []transform.ActorStateHandler, consumers []load.Handler) (TipSetTransformer, ActorTransformer, Loader, error) {
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

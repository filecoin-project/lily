package transform

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	evntbus "github.com/mustafaturan/bus/v3"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/indexer/v2/bus"
	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	v2 "github.com/filecoin-project/lily/model/v2"
)

var log = logging.Logger("transform")

type Kind string

type Result interface {
	Kind() Kind
	Data() interface{}
}

type TipSetStateHandler interface {
	Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan Result) error
	Name() string
	ModelType() v2.ModelMeta
	Matcher() string
}

func NewTipSetStateRouter(reporter string, handlers ...TipSetStateHandler) (*TipSetStateRouter, error) {
	b, err := bus.NewBus()
	if err != nil {
		return nil, err
	}

	handlerChans := make([]chan *extract.TipSetStateResult, len(handlers))
	routerHandlers := make([]TipSetStateHandler, len(handlers))
	registry := make(map[v2.ModelMeta][]TipSetStateHandler)
	for i, handler := range handlers {
		// map of model types to handlers for said type
		registry[handler.ModelType()] = append(registry[handler.ModelType()], handler)
		// maintain list of handlers
		routerHandlers[i] = handler
		// initialize handler channel
		handlerChans[i] = make(chan *extract.TipSetStateResult, 256)
		// register handler for its required model, all models the hander can process are sent on its channel
		hch := handlerChans[i]
		b.Bus.RegisterTopics(handler.ModelType().String())
		b.Bus.RegisterHandler(handler.Name(), evntbus.Handler{
			Handle: func(ctx context.Context, e evntbus.Event) {
				hch <- e.Data.(*extract.TipSetStateResult)
			},
			Matcher: handler.Matcher(),
		})
	}
	return &TipSetStateRouter{
		registry:        registry,
		bus:             b,
		resultCh:        make(chan Result, 1024),
		handlerChannels: handlerChans,
		handlerGrp:      &errgroup.Group{},
		handlers:        routerHandlers,
		reporter:        reporter,
	}, nil
}

type TipSetStateRouter struct {
	registry        map[v2.ModelMeta][]TipSetStateHandler
	bus             *bus.Bus
	resultCh        chan Result
	handlerChannels []chan *extract.TipSetStateResult
	handlerGrp      *errgroup.Group
	handlers        []TipSetStateHandler
	count           int64
	reporter        string
}

func (r *TipSetStateRouter) Start(ctx context.Context) {
	log.Infow("starting router", "topics", r.bus.Bus.Topics())
	for i, handler := range r.handlers {
		log.Infow("start handler", "type", handler.Name())
		i := i
		handler := handler
		r.handlerGrp.Go(func() error {
			return handler.Run(ctx, r.reporter, r.handlerChannels[i], r.resultCh)
		})
	}
}

func (r *TipSetStateRouter) Stop() error {
	log.Info("stopping router")
	// close all channel feeding handlers
	for _, c := range r.handlerChannels {
		close(c)
	}
	log.Info("closed handler channels")
	// wait for handlers to complete and drain their now closed channel
	err := r.handlerGrp.Wait()
	if err != nil {
		log.Info("handlers failed to complete", "error", err)
	}
	log.Info("handlers completed successfully")
	// close the output channel signaling there are no more results to handle.
	close(r.resultCh)
	log.Infow("router stopped", "count", r.count)
	return err
}

func (r *TipSetStateRouter) Route(ctx context.Context, data *extract.TipSetStateResult) error {
	r.count++
	log.Debugw("routing data", "type", data.Task.String())
	return r.bus.Bus.Emit(ctx, data.Task.String(), data)
}

func (r *TipSetStateRouter) Results() chan Result {
	return r.resultCh
}

type ActorStateHandler interface {
	Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan Result) error
	Name() string
	ModelType() v2.ModelMeta
	Matcher() string
}

func NewActorStateRouter(reporter string, handlers ...ActorStateHandler) (*ActorStateRouter, error) {
	b, err := bus.NewBus()
	if err != nil {
		return nil, err
	}

	handlerChans := make([]chan *extract.ActorStateResult, len(handlers))
	routerHandlers := make([]ActorStateHandler, len(handlers))
	registry := make(map[v2.ModelMeta][]ActorStateHandler)
	for i, handler := range handlers {
		// map of model types to handlers for said type
		registry[handler.ModelType()] = append(registry[handler.ModelType()], handler)
		// maintain list of handlers
		routerHandlers[i] = handler
		// initialize handler channel
		handlerChans[i] = make(chan *extract.ActorStateResult, 256)
		// register handler for its required model, all models the hander can process are sent on its channel
		hch := handlerChans[i]
		b.Bus.RegisterTopics(handler.ModelType().String())
		b.Bus.RegisterHandler(handler.Name(), evntbus.Handler{
			Handle: func(ctx context.Context, e evntbus.Event) {
				hch <- e.Data.(*extract.ActorStateResult)
			},
			Matcher: handler.Matcher(),
		})
	}
	return &ActorStateRouter{
		registry:        registry,
		bus:             b,
		resultCh:        make(chan Result, 1024),
		handlerChannels: handlerChans,
		handlerGrp:      &errgroup.Group{},
		handlers:        routerHandlers,
		reporter:        reporter,
	}, nil
}

type ActorStateRouter struct {
	registry        map[v2.ModelMeta][]ActorStateHandler
	bus             *bus.Bus
	resultCh        chan Result
	handlerChannels []chan *extract.ActorStateResult
	handlerGrp      *errgroup.Group
	handlers        []ActorStateHandler
	count           int64
	reporter        string
}

func (r *ActorStateRouter) Start(ctx context.Context) {
	log.Infow("starting router", "topics", r.bus.Bus.Topics())
	for i, handler := range r.handlers {
		log.Infow("start handler", "type", handler.Name())
		i := i
		handler := handler
		r.handlerGrp.Go(func() error {
			return handler.Run(ctx, r.reporter, r.handlerChannels[i], r.resultCh)
		})
	}
}

func (r *ActorStateRouter) Stop() error {
	log.Info("stopping router")
	// close all channel feeding handlers
	for _, c := range r.handlerChannels {
		close(c)
	}
	log.Info("closed handler channels")
	// wait for handlers to complete and drain their now closed channel
	err := r.handlerGrp.Wait()
	if err != nil {
		log.Info("handlers failed to complete", "error", err)
	}
	log.Info("handlers completed successfully")
	// close the output channel signaling there are no more results to handle.
	close(r.resultCh)
	log.Infow("router stopped", "count", r.count)
	return err
}

func (r *ActorStateRouter) Route(ctx context.Context, data *extract.ActorStateResult) error {
	r.count++
	log.Debugw("routing data", "type", data.Task.String())
	return r.bus.Bus.Emit(ctx, data.Task.String(), data)
}

func (r *ActorStateRouter) Results() chan Result {
	return r.resultCh
}

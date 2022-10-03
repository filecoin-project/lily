package load

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	evntbus "github.com/mustafaturan/bus/v3"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/indexer/v2/bus"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
)

var log = logging.Logger("load")

type Handler interface {
	Consume(ctx context.Context, in chan transform.Result) error
	Name() string
	Type() transform.Kind
}

type Router struct {
	registry        map[transform.Kind][]Handler
	bus             *bus.Bus
	handlerChannels []chan transform.Result
	handlerGrp      *errgroup.Group
	handlers        []Handler
}

func NewRouter(handlers ...Handler) (*Router, error) {
	b, err := bus.NewBus()
	if err != nil {
		return nil, err
	}
	handlerChans := make([]chan transform.Result, len(handlers))
	routerHandlers := make([]Handler, len(handlers))
	registry := make(map[transform.Kind][]Handler)
	for i, handler := range handlers {
		// map of handler types to handlers for said types.
		registry[handler.Type()] = append(registry[handler.Type()], handler)
		// list of all handlers
		routerHandlers = append(routerHandlers, handler)
		// init handler channel
		handlerChans[i] = make(chan transform.Result, 8) // TODO buffer
		//register handler topic with bus
		b.Bus.RegisterTopics(string(handler.Type()))
		hch := handlerChans[i]
		// register handler for its required model, all models the hander can process are sent on its channel
		b.Bus.RegisterHandler(handler.Name(), evntbus.Handler{
			Handle: func(ctx context.Context, e evntbus.Event) {
				hch <- e.Data.(transform.Result)
			},
			Matcher: string(handler.Type()),
		})
	}
	return &Router{
		registry:        registry,
		bus:             b,
		handlerChannels: handlerChans,
		handlerGrp:      &errgroup.Group{},
		handlers:        handlers,
	}, nil
}

func (r *Router) Start(ctx context.Context) {
	log.Infow("starting router", "topics", r.bus.Bus.Topics())
	for i, handler := range r.handlers {
		log.Infow("start handler", "type", handler.Type())
		i := i
		handler := handler
		r.handlerGrp.Go(func() error {
			return handler.Consume(ctx, r.handlerChannels[i])
		})
	}
}

func (r *Router) Stop() error {
	log.Info("stopping router")
	for _, c := range r.handlerChannels {
		close(c)
	}
	log.Info("closed handler channels")
	err := r.handlerGrp.Wait()
	log.Infow("handlers completed", "error", err)
	log.Info("router stopped")
	return err
}

func (r *Router) Route(ctx context.Context, data transform.Result) error {
	log.Debugw("routing data", "type", data.Kind())
	return r.bus.Bus.Emit(ctx, string(data.Kind()), data)
}

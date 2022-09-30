package load

import (
	"context"
	"sync"

	evntbus "github.com/mustafaturan/bus/v3"

	"github.com/filecoin-project/lily/chain/indexer/v2/bus"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
)

type Handler interface {
	Consume(ctx context.Context, wg *sync.WaitGroup, in chan transform.Result)
	Type() transform.Kind
}

type Router struct {
	registry        map[transform.Kind][]Handler
	bus             *bus.Bus
	handlerChannels []chan transform.Result
	handlerWg       *sync.WaitGroup
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
		handlerChans[i] = make(chan transform.Result) // TODO buffer
		//register handler topic with bus
		b.Bus.RegisterTopics(string(handler.Type()))
		hch := handlerChans[i]
		// register handler for its required model, all models the hander can process are sent on its channel
		b.Bus.RegisterHandler(string(handler.Type()), evntbus.Handler{
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
		handlerWg:       &sync.WaitGroup{},
		handlers:        handlers,
	}, nil
}

func (r *Router) Start(ctx context.Context) {
	for i, handler := range r.handlers {
		r.handlerWg.Add(1)
		go handler.Consume(ctx, r.handlerWg, r.handlerChannels[i])
	}
}

func (r *Router) Stop() {
	for _, c := range r.handlerChannels {
		close(c)
	}
	r.handlerWg.Wait()
}

func (r *Router) Route(ctx context.Context, data transform.Result) error {
	return r.bus.Bus.Emit(ctx, string(data.Kind()), data)
}

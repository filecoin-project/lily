package transform

import (
	"context"
	"sync"

	"github.com/filecoin-project/lotus/chain/types"
	evntbus "github.com/mustafaturan/bus/v3"

	"github.com/filecoin-project/lily/chain/indexer/v2/bus"
	v22 "github.com/filecoin-project/lily/chain/indexer/v2/extract"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

type Kind string

type Result interface {
	Kind() Kind
	Data() interface{}
}

type IndexState interface {
	Task() v2.ModelMeta
	Current() *types.TipSet
	Executed() *types.TipSet
	Complete() bool
	State() *v22.StateResult
}

type Handler interface {
	Run(ctx context.Context, wg *sync.WaitGroup, api tasks.DataSource, in chan IndexState, out chan Result)
	ModelType() v2.ModelMeta
}

func NewRouter(handlers ...Handler) (*Router, error) {
	b, err := bus.NewBus()
	if err != nil {
		return nil, err
	}
	handlerChans := make([]chan IndexState, len(handlers))
	routerHandlers := make([]Handler, len(handlers))
	registry := make(map[v2.ModelMeta][]Handler)
	for i, handler := range handlers {
		// map of model types to handlers for said type
		registry[handler.ModelType()] = append(registry[handler.ModelType()], handler)
		// maintain list of handlers
		routerHandlers[i] = handler
		// initialize handler channel
		handlerChans[i] = make(chan IndexState) // TODO buffer channel
		// register the handle topic with the bus
		b.Bus.RegisterTopics(handler.ModelType().String())
		// register handler for its required model, all models the hander can process are sent on its channel
		hch := handlerChans[i]
		b.Bus.RegisterHandler(handler.ModelType().String(), evntbus.Handler{
			Handle: func(ctx context.Context, e evntbus.Event) {
				hch <- e.Data.(IndexState)
			},
			Matcher: handler.ModelType().String(),
		})
	}
	return &Router{
		registry:        registry,
		bus:             b,
		resultCh:        make(chan Result), // TODO buffer channel
		handlerChannels: handlerChans,
		handlerWg:       &sync.WaitGroup{},
		handlers:        routerHandlers,
	}, nil
}

type Router struct {
	registry        map[v2.ModelMeta][]Handler
	bus             *bus.Bus
	resultCh        chan Result
	handlerChannels []chan IndexState
	handlerWg       *sync.WaitGroup
	handlers        []Handler
}

func (r *Router) Start(ctx context.Context, api tasks.DataSource) {
	for i, handler := range r.handlers {
		r.handlerWg.Add(1)
		go handler.Run(ctx, r.handlerWg, api, r.handlerChannels[i], r.resultCh)
	}
}

func (r *Router) Stop() {
	// close all channel feeding handlers
	for _, c := range r.handlerChannels {
		close(c)
	}
	// wait for handlers to complete and drain their now closed channel
	r.handlerWg.Wait()
	// close the output channel signaling there are no more results to handle.
	close(r.resultCh)
}

func (r *Router) Route(ctx context.Context, data IndexState) error {
	return r.bus.Bus.Emit(ctx, data.Task().String(), data)
}

func (r *Router) Results() chan Result {
	return r.resultCh
}

func (r *Router) registerHandler(in chan IndexState, matcher string) {
	r.handlerChannels = append(r.handlerChannels, in)
	r.bus.Bus.RegisterHandler(matcher, evntbus.Handler{
		Handle: func(ctx context.Context, e evntbus.Event) {
			in <- e.Data.(IndexState)
		},
		Matcher: matcher,
	})
}

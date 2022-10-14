package transform

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	evntbus "github.com/mustafaturan/bus/v3"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/indexer/v2/bus"
	v2 "github.com/filecoin-project/lily/model/v2"
)

var log = logging.Logger("transform")

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
	Models() []v2.LilyModel
	ExtractionState() interface{}
}

type Handler interface {
	Run(ctx context.Context, in chan IndexState, out chan Result) error
	Name() string
	ModelType() v2.ModelMeta
	Matcher() string
}

func NewRouter(topics []v2.ModelMeta, handlers ...Handler) (*Router, error) {
	b, err := bus.NewBus()
	if err != nil {
		return nil, err
	}

	// register topics with the bus
	for _, t := range topics {
		b.Bus.RegisterTopics(t.String())
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
		handlerChans[i] = make(chan IndexState, 256)
		// register handler for its required model, all models the hander can process are sent on its channel
		hch := handlerChans[i]
		b.Bus.RegisterHandler(handler.Name(), evntbus.Handler{
			Handle: func(ctx context.Context, e evntbus.Event) {
				hch <- e.Data.(IndexState)
			},
			Matcher: handler.Matcher(),
		})
	}
	return &Router{
		registry:        registry,
		bus:             b,
		resultCh:        make(chan Result, 1024),
		handlerChannels: handlerChans,
		handlerGrp:      &errgroup.Group{},
		handlers:        routerHandlers,
	}, nil
}

type Router struct {
	registry        map[v2.ModelMeta][]Handler
	bus             *bus.Bus
	resultCh        chan Result
	handlerChannels []chan IndexState
	handlerGrp      *errgroup.Group
	handlers        []Handler
	count           int64
}

func (r *Router) Start(ctx context.Context) {
	log.Infow("starting router", "topics", r.bus.Bus.Topics())
	for i, handler := range r.handlers {
		log.Infow("start handler", "type", handler.Name())
		i := i
		handler := handler
		r.handlerGrp.Go(func() error {
			return handler.Run(ctx, r.handlerChannels[i], r.resultCh)
		})
	}
}

func (r *Router) Stop() error {
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

func (r *Router) Route(ctx context.Context, data IndexState) error {
	r.count++
	log.Debugw("routing data", "type", data.Task().String())
	return r.bus.Bus.Emit(ctx, data.Task().String(), data)
}

func (r *Router) Results() chan Result {
	return r.resultCh
}

package v2

import (
	"context"
	"sync"

	evntbus "github.com/mustafaturan/bus/v3"

	"github.com/filecoin-project/lily/chain/indexer/v2/bus"
	"github.com/filecoin-project/lily/model"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

type HandlerResultType string

type HandlerResult interface {
	Type() HandlerResultType
	Data() interface{}
}

type Handler interface {
	Run(ctx context.Context, wg *sync.WaitGroup, api tasks.DataSource, in chan *TipSetResult, out chan HandlerResult)
	ModelType() v2.ModelMeta
}

// TODO global

func NewHandlerRouter() (*HandlerRouter, error) {
	b, err := bus.NewBus()
	if err != nil {
		return nil, err
	}
	return &HandlerRouter{
		Bus:             b,
		HandlerRegistry: make(map[v2.ModelMeta][]Handler),
		HandlerWg:       &sync.WaitGroup{},
	}, nil
}

type HandlerRouter struct {
	HandlerRegistry map[v2.ModelMeta][]Handler
	Bus             *bus.Bus
	outCh           chan HandlerResult
	HandlerChannels []chan *TipSetResult
	HandlerWg       *sync.WaitGroup
}

func (hr *HandlerRouter) AddHandler(h Handler) {
	hr.RegisterTopics(h.ModelType())
	hr.HandlerRegistry[h.ModelType()] = append(hr.HandlerRegistry[h.ModelType()], h)
}

func (hr *HandlerRouter) Start(ctx context.Context, api tasks.DataSource) {
	outChan := make(chan HandlerResult)
	for meta, handlers := range hr.HandlerRegistry {
		inChan := make(chan *TipSetResult)
		hr.RegisterHandler(inChan, meta.String())
		for _, handler := range handlers {
			hr.HandlerWg.Add(1)
			go handler.Run(ctx, hr.HandlerWg, api, inChan, outChan)
		}
	}
	hr.outCh = outChan
}

func (hr *HandlerRouter) Stop() {
	// close all channel feeding handlers
	for _, c := range hr.HandlerChannels {
		close(c)
	}
	// wait for handlers to complete and drain their now closed channel
	hr.HandlerWg.Wait()
	// close the output channel signaling there are no more results to handle.
	close(hr.outCh)
}

func (hr *HandlerRouter) RegisterTopics(meta v2.ModelMeta) {
	hr.Bus.Bus.RegisterTopics(meta.String())
}

func (hr *HandlerRouter) RegisterHandler(in chan *TipSetResult, matcher string) {
	hr.HandlerChannels = append(hr.HandlerChannels, in)
	hr.Bus.Bus.RegisterHandler(matcher, evntbus.Handler{
		Handle: func(ctx context.Context, e evntbus.Event) {
			in <- e.Data.(*TipSetResult)
		},
		Matcher: matcher,
	})
}

func (hr *HandlerRouter) Emit(ctx context.Context, data *TipSetResult) error {
	return hr.Bus.Bus.Emit(ctx, data.Task.String(), data)
}

func (hr *HandlerRouter) Results() chan HandlerResult {
	return hr.outCh
}

type ResultConsumer interface {
	Consume(ctx context.Context, wg *sync.WaitGroup, in chan HandlerResult)
	Type() HandlerResultType
}

func NewResultRouter() (*ResultRouter, error) {
	b, err := bus.NewBus()
	if err != nil {
		return nil, err
	}
	return &ResultRouter{
		ResultRegistry: make(map[HandlerResultType][]ResultConsumer),
		Bus:            b,
		ConsumerWg:     &sync.WaitGroup{},
	}, nil
}

type ResultRouter struct {
	ResultChannels []chan HandlerResult
	ResultRegistry map[HandlerResultType][]ResultConsumer
	Bus            *bus.Bus
	ConsumerWg     *sync.WaitGroup
}

func (rr *ResultRouter) AddConsumer(c ResultConsumer) {
	rr.Bus.Bus.RegisterTopics(string(c.Type()))
	rr.ResultRegistry[c.Type()] = append(rr.ResultRegistry[c.Type()], c)
}

func (rr *ResultRouter) Start(ctx context.Context) {
	for resType, consumers := range rr.ResultRegistry {
		inChan := make(chan HandlerResult)
		rr.ResultChannels = append(rr.ResultChannels, inChan)
		rr.Bus.Bus.RegisterHandler(string(resType), evntbus.Handler{
			Handle: func(ctx context.Context, e evntbus.Event) {
				inChan <- e.Data.(HandlerResult)
			},
			Matcher: string(resType),
		})
		for _, consumer := range consumers {
			rr.ConsumerWg.Add(1)
			go consumer.Consume(ctx, rr.ConsumerWg, inChan)
		}
	}
}

func (rr *ResultRouter) Emit(ctx context.Context, data HandlerResult) error {
	return rr.Bus.Bus.Emit(ctx, string(data.Type()), data)
}

func (rr *ResultRouter) Stop() {
	// close channels feeding consumers to signal there are no more results to consume.
	for _, c := range rr.ResultChannels {
		close(c)
	}
	// wait for consumers to finish draining their channels
	rr.ConsumerWg.Wait()
}

type PersistableResultConsumer struct {
	strg model.Storage
}

func (p *PersistableResultConsumer) Type() HandlerResultType {
	return "persistable"
}

func (p *PersistableResultConsumer) Consume(ctx context.Context, wg *sync.WaitGroup, in chan HandlerResult) {
	defer wg.Done()
	for res := range in {
		select {
		case <-ctx.Done():
			return
		default:
			log.Infow("consume", "type", res.Type())
			if err := p.strg.PersistBatch(ctx, res.Data().(model.Persistable)); err != nil {
				panic(err)
			}
		}
	}
	log.Info("consumer exit")
}

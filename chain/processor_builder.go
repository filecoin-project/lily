package chain

import (
	"github.com/filecoin-project/lily/lens/task"
	"github.com/filecoin-project/lily/tasks/indexer"
)

func NewStateProcessorBuilder(api task.TaskAPI, name string) *StateProcessorBuilder {
	return &StateProcessorBuilder{api: api, name: name}
}

type StateProcessorBuilder struct {
	options []func(sp *StateProcessor)
	api     task.TaskAPI
	name    string
}

func (spb *StateProcessorBuilder) add(cb func(sp *StateProcessor)) {
	spb.options = append(spb.options, cb)
}

func (spb *StateProcessorBuilder) WithTipSetProcessors(opt map[string]TipSetProcessor) *StateProcessorBuilder {
	spb.add(func(sp *StateProcessor) {
		sp.tipsetProcessors = opt
	})
	return spb
}

func (spb *StateProcessorBuilder) WithTipSetsProcessors(opt map[string]TipSetsProcessor) *StateProcessorBuilder {
	spb.add(func(sp *StateProcessor) {
		sp.tipsetsProcessors = opt
	})
	return spb
}

func (spb *StateProcessorBuilder) WithActorProcessors(opt map[string]ActorProcessor) *StateProcessorBuilder {
	spb.add(func(sp *StateProcessor) {
		sp.actorProcessors = opt
	})
	return spb
}

func (spb *StateProcessorBuilder) Build() *StateProcessor {
	api := spb.api

	sp := &StateProcessor{
		name: spb.name,
		builtinProcessors: map[string]BuiltinProcessor{
			"builtin": indexer.NewTask(api),
		},
		api: spb.api,
	}

	for _, opt := range spb.options {
		opt(sp)
	}

	// build the taskName list
	for name := range sp.builtinProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	for name := range sp.tipsetProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	for name := range sp.tipsetsProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	for name := range sp.actorProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	return sp
}

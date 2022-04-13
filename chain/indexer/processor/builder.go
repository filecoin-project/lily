package processor

import (
	"github.com/filecoin-project/lily/tasks"
)

// NewBuilder returns a Builder used to construct a StateProcessor
func NewBuilder(api tasks.DataSource, processorName string) *Builder {
	return &Builder{api: api, name: processorName}
}

type Builder struct {
	options []func(sp *StateProcessor)
	api     tasks.DataSource
	name    string
}

func (spb *Builder) add(cb func(sp *StateProcessor)) {
	spb.options = append(spb.options, cb)
}

func (spb *Builder) WithTipSetProcessors(opt map[string]TipSetProcessor) *Builder {
	spb.add(func(sp *StateProcessor) {
		sp.tipsetProcessors = opt
	})
	return spb
}

func (spb *Builder) WithTipSetsProcessors(opt map[string]TipSetsProcessor) *Builder {
	spb.add(func(sp *StateProcessor) {
		sp.tipsetsProcessors = opt
	})
	return spb
}

func (spb *Builder) WithActorProcessors(opt map[string]ActorProcessor) *Builder {
	spb.add(func(sp *StateProcessor) {
		sp.actorProcessors = opt
	})
	return spb
}

func (spb *Builder) WithBuiltinProcessors(opt map[string]ReportProcessor) *Builder {
	spb.add(func(sp *StateProcessor) {
		sp.builtinProcessors = opt
	})
	return spb
}

func (spb *Builder) Build() *StateProcessor {
	// build the taskName list
	sp := &StateProcessor{
		name: spb.name,
		api:  spb.api,
	}

	for _, opt := range spb.options {
		opt(sp)
	}

	return sp
}

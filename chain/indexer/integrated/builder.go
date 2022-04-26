package integrated

import "github.com/filecoin-project/lily/tasks"

func NewBuilder(node tasks.DataSource, name string) *Builder {
	return &Builder{api: node, name: name}
}

type Builder struct {
	options []func(ti *TipSetIndexer)
	api     tasks.DataSource
	name    string
}

func (b *Builder) add(cb func(ti *TipSetIndexer)) {
	b.options = append(b.options, cb)
}

func (b *Builder) WithTasks(tasks []string) *Builder {
	b.add(func(ti *TipSetIndexer) {
		ti.taskNames = tasks
	})
	return b
}

func (b *Builder) Build() (*TipSetIndexer, error) {
	ti := &TipSetIndexer{
		name: b.name,
		node: b.api,
	}

	for _, opt := range b.options {
		opt(ti)
	}

	if err := ti.init(); err != nil {
		return nil, err
	}
	return ti, nil
}

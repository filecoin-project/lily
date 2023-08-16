package tipset

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/tasks"
)

type IndexerBuilder interface {
	WithTasks(tasks []string) IndexerBuilder
	WithInterval(interval int) IndexerBuilder
	Build() (Indexer, error)
	Name() string
}

type Indexer interface {
	TipSet(ctx context.Context, ts *types.TipSet) (chan *Result, chan error, error)
}

var _ IndexerBuilder = (*Builder)(nil)

func NewBuilder(node tasks.DataSource, name string) IndexerBuilder {
	return &Builder{api: node, name: name}
}

type Builder struct {
	options []func(ti *TipSetIndexer)
	api     tasks.DataSource
	name    string
}

func (b *Builder) Name() string {
	return b.name
}

func (b *Builder) add(cb func(ti *TipSetIndexer)) {
	b.options = append(b.options, cb)
}

func (b *Builder) WithTasks(tasks []string) IndexerBuilder {
	b.add(func(ti *TipSetIndexer) {
		ti.taskNames = tasks
	})
	return b
}

func (b *Builder) WithInterval(interval int) IndexerBuilder {
	b.add(func(ti *TipSetIndexer) {
		ti.Interval = interval
	})
	return b
}

func (b *Builder) Build() (Indexer, error) {
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

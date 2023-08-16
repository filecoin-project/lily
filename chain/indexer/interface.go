package indexer

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/lotus/chain/types"
)

// Option specifies the index processing behavior. The interface allows implementations of the Indexer interface
// to be configured independently without changing the declaration of the Indexer.TipSet method.
type Option interface {
	// String returns a string representation of the option.
	String() string

	// Type describes the type of the option.
	Type() OptionType

	// Value returns a value used to create this option.
	Value() interface{}
}

type OptionType int

const (
	IndexTypeOpt OptionType = iota
	TasksOpt
	IntervalOpt
)

type (
	indexTypeOption int
	tasksTypeOption []string
	intervalOption  int
)

// WithTasks returns and Option that specifies the tasks to be indexed.
// It is used by both the distributed and integrated indexers.
func WithTasks(tasks []string) Option {
	return tasksTypeOption(tasks)
}

func (t tasksTypeOption) String() string     { return fmt.Sprintf("Tasks(%s)", strings.Join(t, ",")) }
func (t tasksTypeOption) Type() OptionType   { return TasksOpt }
func (t tasksTypeOption) Value() interface{} { return []string(t) }

type IndexerType int

func (i IndexerType) String() string {
	switch i {
	case Undefined:
		return "undefined"
	case Watch:
		return "watch"
	case Walk:
		return "walk"
	case Index:
		return "index"
	case Fill:
		return "fill"
	default:
		panic(fmt.Sprintf("developer error unknown indexer type: %d", i))
	}
}

// WithIndexerType returns and Option that specifies the type of index operation being performed.
// It is used by the distributed indexer to determine priority of the TipSet being indexed.
func WithIndexerType(it IndexerType) Option {
	return indexTypeOption(it)
}

const (
	Undefined IndexerType = iota
	Watch
	Walk
	Index
	Fill
)

func (o indexTypeOption) String() string     { return fmt.Sprintf("IndexerType(%d)", o) }
func (o indexTypeOption) Type() OptionType   { return IndexTypeOpt }
func (o indexTypeOption) Value() interface{} { return IndexerType(o) }

func WithInterval(interval int) Option {
	return intervalOption(interval)
}

func (o intervalOption) String() string     { return fmt.Sprintf("Interval: %d", o) }
func (o intervalOption) Type() OptionType   { return IntervalOpt }
func (o intervalOption) Value() interface{} { return o }

// IndexerOptions are used by implementations of the Indexer interface for configuration.
type IndexerOptions struct {
	IndexType IndexerType
	Tasks     []string
	Interval  int
}

// ConstructOptions returns an IndexerOptions struct that may be used to configured implementations of the Indexer interface.
func ConstructOptions(opts ...Option) (IndexerOptions, error) {
	res := IndexerOptions{
		IndexType: Undefined,
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case indexTypeOption:
			res.IndexType = IndexerType(o)
		case tasksTypeOption:
			res.Tasks = []string(o)
			if len(res.Tasks) == 0 {
				return IndexerOptions{}, fmt.Errorf("tasks options cannot be empty")
			}
		case intervalOption:
			res.Interval = int(o)
		default:
		}
	}
	return res, nil
}

// Indexer implemented to index TipSets.
type Indexer interface {
	// TipSet indexes a TipSet. The returned error is non-nill if a fatal error is encountered. True is returned if the
	// TipSet is indexed successfully, false if returned if the TipSet was only partially indexer.
	TipSet(ctx context.Context, ts *types.TipSet, opts ...Option) (bool, error)
}

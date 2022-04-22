package indexer

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"
)

// Option specifies the task processing behavior.
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
)

type (
	indexTypeOption int
	tasksTypeOption []string
)

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

type IndexerOptions struct {
	IndexType IndexerType
	Tasks     []string
}

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
				return IndexerOptions{}, xerrors.Errorf("tasks options cannot be empty")
			}
		default:
		}
	}
	return res, nil
}

type Indexer interface {
	TipSet(ctx context.Context, ts *types.TipSet, opts ...Option) (bool, error)
}

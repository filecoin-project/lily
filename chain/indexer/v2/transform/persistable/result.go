package persistable

import (
	"time"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/model"
)

type Result struct {
	Model    model.Persistable
	Metadata *Meta
}

func (p *Result) Meta() interface{} {
	return p.Metadata
}

type Meta struct {
	TipSet    *types.TipSet
	Name      string
	Errors    []error
	StartTime time.Time
	EndTime   time.Time
}

func (p *Result) Kind() transform.Kind {
	return "persistable"
}

func (p *Result) Data() interface{} {
	return p.Model
}

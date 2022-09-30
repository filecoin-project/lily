package persistable

import (
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/model"
)

type Result struct {
	Model model.Persistable
}

func (p *Result) Kind() transform.Kind {
	return "persistable"
}

func (p *Result) Data() interface{} {
	return p.Model
}

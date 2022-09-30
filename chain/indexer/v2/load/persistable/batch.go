package persistable

import (
	"context"
	"sync"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/model"
)

type PersistableResultConsumer struct {
	Strg model.Storage
}

func (p *PersistableResultConsumer) Type() transform.Kind {
	return "persistable"
}

func (p *PersistableResultConsumer) Consume(ctx context.Context, wg *sync.WaitGroup, in chan transform.Result) {
	defer wg.Done()
	for res := range in {
		select {
		case <-ctx.Done():
			return
		default:
			if err := p.Strg.PersistBatch(ctx, res.Data().(model.Persistable)); err != nil {
				panic(err)
			}
		}
	}
}

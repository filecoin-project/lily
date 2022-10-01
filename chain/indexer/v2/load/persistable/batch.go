package persistable

import (
	"context"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/model"
)

type PersistableResultConsumer struct {
	Strg model.Storage
}

func (p *PersistableResultConsumer) Type() transform.Kind {
	return "persistable"
}

func (p *PersistableResultConsumer) Consume(ctx context.Context, in chan transform.Result) error {
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if res.Data() == nil {
				continue
			}
			if err := p.Strg.PersistBatch(ctx, res.Data().(model.Persistable)); err != nil {
				return err
			}
		}
	}
	return nil
}

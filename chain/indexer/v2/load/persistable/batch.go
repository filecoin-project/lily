package persistable

import (
	"context"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/model"
)

type PersistableResultConsumer struct {
	Strg model.Storage
}

func (p *PersistableResultConsumer) Type() transform.Kind {
	return "persistable"
}

func (p *PersistableResultConsumer) Name() string {
	info := PersistableResultConsumer{}
	return reflect.TypeOf(info).Name()
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
			if l, ok := res.Data().(model.PersistableList); ok && len(l) == 0 {
				continue
			}
			if err := p.Strg.PersistBatch(ctx, res.Data().(model.Persistable)); err != nil {
				return err
			}
		}
	}
	return nil
}

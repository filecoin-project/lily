package persistable

import (
	"context"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/model"
)

var log = logging.Logger("persistable/batch")

type PersistableResultConsumer struct {
	Strg    model.Storage
	GetName func(string) string
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
			l, ok := res.Data().(model.Persistable)
			if !ok {
				log.Errorw("failed to reflect consumer data")
			}
			if err := p.Strg.PersistBatch(ctx, l); err != nil {
				return err
			}
		}
	}
	return nil
}

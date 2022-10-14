package persistable

import (
	"context"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

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
			var persist []model.Persistable
			meta, ok := res.Meta().(*persistable.Meta)
			if ok && meta != nil {
				status := visormodel.ProcessingStatusOK
				if meta.Errors != nil && len(meta.Errors) > 0 {
					status = visormodel.ProcessingStatusError
				}
				report := &visormodel.ProcessingReport{
					Height:            int64(meta.TipSet.Height()),
					StateRoot:         meta.TipSet.ParentState().String(),
					Reporter:          "TODO",
					Task:              p.GetName(meta.Name),
					StartedAt:         meta.StartTime,
					CompletedAt:       meta.EndTime,
					Status:            status,
					StatusInformation: "",
					ErrorsDetected:    meta.Errors,
				}
				persist = append(persist, report)
			}
			if res.Data() != nil {
				if l, ok := res.Data().(model.Persistable); ok {
					persist = append(persist, l)
				}
			}
			if err := p.Strg.PersistBatch(ctx, model.PersistableList(persist)); err != nil {
				return err
			}
		}
	}
	return nil
}

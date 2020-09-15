package processor

import (
	"context"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/storage"
)

func NewPublisher(s *storage.Database) *Publisher {
	return &Publisher{
		storage: s,
		log:     logging.Logger("publisher"),
	}
}

type Publisher struct {
	storage *storage.Database
	log     *logging.ZapEventLogger
}

func (p *Publisher) Publish(ctx context.Context, payload model.Persistable) error {
	// TODO explore use of channel.
	// TODO buffer and use routine.
	return payload.Persist(ctx, p.storage.DB)
}

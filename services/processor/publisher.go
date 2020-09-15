package processor

import (
	"context"
	miner2 "github.com/filecoin-project/visor/model/actors/miner"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/visor/storage"
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

func (p *Publisher) Publish(ctx context.Context, payload interface{}) error {
	// TODO buffer the writes to the database and use statements where possible.
	switch v := payload.(type) {
	case *miner2.MinerTaskResult:
		if err := v.Persist(ctx, p.storage.DB); err != nil {
			return err
		}
	}
	return nil
}

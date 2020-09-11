package processor

import (
	"context"

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
	switch v := payload.(type) {
	case *MinerProcessResult:
		// TODO actually store the thing
		p.log.Errorw("Storing MinerProcessResult", "address", v.addr.String())
		// do stuff
	}
	return nil
}

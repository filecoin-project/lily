package processor

import (
	"context"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/storage"
)

func NewPublisher(s *storage.Database, pubCh <-chan model.Persistable) *Publisher {
	return &Publisher{
		storage: s,
		pubCh:   pubCh,
		log:     logging.Logger("publisher"),
	}
}

type Publisher struct {
	storage *storage.Database
	pubCh   <-chan model.Persistable
	log     *logging.ZapEventLogger
}

func (p *Publisher) Start(ctx context.Context) {
	p.log.Info("starting publisher")
	go func() {
		for {
			select {
			case <-ctx.Done():
				p.log.Info("stopping publisher")
				return
			case persistable := <-p.pubCh:
				go func() {
					if err := persistable.Persist(ctx, p.storage.DB); err != nil {
						// TODO handle this case with a retry
						p.log.Error("persisting", "error", err.Error())
					}
				}()
			}
		}
	}()

}

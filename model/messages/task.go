package messages

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MessageTaskResult struct {
	Messages          Messages
	ParsedMessages    ParsedMessages
	BlockMessages     BlockMessages
	Receipts          Receipts
	MessageGasEconomy *MessageGasEconomy
}

func (mtr *MessageTaskResult) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	if err := mtr.Messages.Persist(ctx, s, version); err != nil {
		return err
	}
	if err := mtr.BlockMessages.Persist(ctx, s, version); err != nil {
		return err
	}
	if err := mtr.Receipts.Persist(ctx, s, version); err != nil {
		return err
	}
	if err := mtr.MessageGasEconomy.Persist(ctx, s, version); err != nil {
		return err
	}
	if err := mtr.ParsedMessages.Persist(ctx, s, version); err != nil {
		return err
	}

	return nil
}

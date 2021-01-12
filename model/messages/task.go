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

func (mtr *MessageTaskResult) Persist(ctx context.Context, s model.StorageBatch) error {
	if err := mtr.Messages.Persist(ctx, s); err != nil {
		return err
	}
	if err := mtr.BlockMessages.Persist(ctx, s); err != nil {
		return err
	}
	if err := mtr.Receipts.Persist(ctx, s); err != nil {
		return err
	}
	if err := mtr.MessageGasEconomy.Persist(ctx, s); err != nil {
		return err
	}
	if err := mtr.ParsedMessages.Persist(ctx, s); err != nil {
		return err
	}

	return nil
}

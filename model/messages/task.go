package messages

import (
	"context"

	"github.com/filecoin-project/lily/model"
)

type MessageTaskResult struct {
	Messages          Messages
	ParsedMessages    ParsedMessages
	BlockMessages     BlockMessages
	Receipts          Receipts
	MessageGasEconomy *MessageGasEconomy
}

func (mtr *MessageTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
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

	err := mtr.ParsedMessages.Persist(ctx, s, version)

	return err
}

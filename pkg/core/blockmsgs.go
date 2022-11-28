package core

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/tasks"
)

type BlockMessage struct {
	Meta *MessageMeta

	// Message
	Message *types.Message
}

func ExtractBlockMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]*BlockMessage, error) {
	blksMsgs, err := api.TipSetBlockMessages(ctx, current)
	if err != nil {
		return nil, err
	}

	var out = make([]*BlockMessage, 0, len(blksMsgs))
	for _, blkMsgs := range blksMsgs {
		for _, msg := range blkMsgs.BlsMessages {
			out = append(out, &BlockMessage{
				Meta: &MessageMeta{
					MessageCID: msg.Cid(),
					SizeBytes:  int64(msg.ChainLength()),
				},
				Message: msg,
			})
		}
		for _, msg := range blkMsgs.SecpMessages {
			out = append(out, &BlockMessage{
				Meta: &MessageMeta{
					MessageCID: msg.Cid(),
					Signature:  &msg.Signature,
					SizeBytes:  int64(msg.ChainLength()),
				},
				Message: msg.VMMessage(),
			})
		}
	}
	return out, nil
}

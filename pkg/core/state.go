package core

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/tasks"
)

type State struct {
	TipSetState     *TipSetState
	BlockMessage    []*BlockMessage
	ExecutedMessage []*ExecutedMessage
	VMMessage       []*VMMessage
	Supply          *Supply
	Actors          ActorChanges
}

func ExtractState(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*State, error) {
	tsState, err := ExtractTipSetState(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}
	blkMsgs, err := ExtractBlockMessages(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}
	exeMsgs, err := ExtractExecutedMessages(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}
	vmMsgs, err := ExtractVMMessages(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}
	supply, err := ExtractTokenSupply(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}
	actors, err := ExtractActorChanges(ctx, api, current, executed)
	if err != nil {
		return nil, err
	}
	// TODO implicit messages

	return &State{
		TipSetState:     tsState,
		BlockMessage:    blkMsgs,
		ExecutedMessage: exeMsgs,
		VMMessage:       vmMsgs,
		Supply:          supply,
		Actors:          actors,
	}, nil
}

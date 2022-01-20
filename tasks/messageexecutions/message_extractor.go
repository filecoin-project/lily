package messageexecutions

import (
	"context"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lily/tasks/messages"
	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"
)

func init() {
	model.RegisterTipSetModelExtractor(&messagemodel.InternalMessage{}, MessageExecutionExtractor{})
	model.RegisterTipSetModelExtractor(&messagemodel.InternalParsedMessage{}, ParsedMessageExecutionExtractor{})
}

var _ model.TipSetStateExtractor = (*MessageExecutionExtractor)(nil)

type MessageExecutionExtractor struct{}

func (MessageExecutionExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	// TODO this is unfortunate, but sticks with the pattern, maybe pass a filter here instead? A similar issues exists with the miner actor.
	res, _, err := process(ctx, current, previous, api)
	return res, err
}

func process(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, model.Persistable, error) {
	// TODO this is the expensive call, note that the method is memoized in the api so should be cheap when called more than once.
	mex, err := api.GetMessageExecutionsForTipSet(ctx, current, previous)
	if err != nil {
		return nil, nil, err
	}

	var (
		internalResult       = make(messagemodel.InternalMessageList, 0, len(mex))
		internalParsedResult = make(messagemodel.InternalParsedMessageList, 0, len(mex))
		errorsDetected       = make([]*messages.MessageError, 0) // we don't know the cap since mex is recursive in nature.
	)

	for _, m := range mex {
		// we don't yet record implicit messages in the other message task, record them here.
		if m.Implicit {
			toName, toFamily, err := util.ActorNameAndFamilyFromCode(m.ToActorCode)
			if err != nil {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid,
					Error: xerrors.Errorf("failed get message to actor name and family: %w", err).Error(),
				})
			}
			internalResult = append(internalResult, &messagemodel.InternalMessage{
				Height:        int64(m.Height),
				Cid:           m.Cid.String(),
				SourceMessage: "", // there is no source for implicit messages, they include cron tick and reward messages only
				StateRoot:     m.StateRoot.String(),
				From:          m.Message.From.String(),
				To:            m.Message.To.String(),
				ActorName:     toName,
				ActorFamily:   toFamily,
				Value:         m.Message.Value.String(),
				Method:        uint64(m.Message.Method),
				ExitCode:      int64(m.Ret.ExitCode),
				GasUsed:       m.Ret.GasUsed,
			})
			method, params, err := util.MethodAndParamsForMessage(m.Message, m.ToActorCode)
			if err != nil {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid,
					Error: xerrors.Errorf("failed parse method and params for message: %w", err).Error(),
				})
			}
			internalParsedResult = append(internalParsedResult, &messagemodel.InternalParsedMessage{
				Height: int64(m.Height),
				Cid:    m.Cid.String(),
				From:   m.Message.From.String(),
				To:     m.Message.To.String(),
				Value:  m.Message.Value.String(),
				Method: method,
				Params: params,
			})
		}
	}
	return internalResult, internalParsedResult, nil
}

func (MessageExecutionExtractor) Name() string {
	return "internal_messages"
}

var _ model.TipSetStateExtractor = (*ParsedMessageExecutionExtractor)(nil)

type ParsedMessageExecutionExtractor struct{}

func (ParsedMessageExecutionExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	// TODO this is unfortunate, but sticks with the pattern, maybe pass a filter here instead? A similar issues exists with the miner actor.
	_, res, err := process(ctx, current, previous, api)
	return res, err
}

func (ParsedMessageExecutionExtractor) Name() string {
	return "internal_parsed_messages"
}

package parsedmessage

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/messages"

	"github.com/filecoin-project/lotus/chain/types"
)

var log = logging.Logger("lily/tasks/parsedmsg")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "parsed_messages"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	grp, _ := errgroup.WithContext(ctx)

	var getActorCodeFn func(ctx context.Context, address address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCodeFn, err = util.MakeGetActorCodeFunc(ctx, t.node.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("getting actor code lookup function: %w", err)
		}
		return nil
	})

	var blkMsgRec []*lens.BlockMessageReceipts
	grp.Go(func() error {
		var err error
		blkMsgRec, err = t.node.TipSetMessageReceipts(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages and receipts: %w", err)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		report.ErrorsDetected = err
		return nil, report, nil
	}

	var (
		parsedMessageResults = make(messagemodel.ParsedMessages, 0)
		errorsDetected       = make([]*messages.MessageError, 0)
		msgSeen              = cid.NewSet()
	)

	for _, msgrec := range blkMsgRec {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		itr, err := msgrec.Iterator()
		if err != nil {
			return nil, nil, err
		}

		for itr.HasNext() {
			m, _, r := itr.Next()

			// if we have already visited this message continue
			if !msgSeen.Visit(m.Cid()) {
				continue
			}

			// if this message failed to apply successfully continue
			if r.ExitCode.IsError() {
				log.Infof("skip parsing message: %v, reason: receipt with exitcode: %s", m.Cid(), r.ExitCode)
				continue
			}

			// since the message applied successfully (non-zero exitcode) its receiver must exist on chain.
			toActorCode, found := getActorCodeFn(ctx, m.VMMessage().To)
			if !found {
				// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
				// message fails then Lotus will not allow the actor to be created and we are left with an address of an
				// unknown type.
				// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
				// indicates that Lily actor type detection is out of date.
				log.Errorw("parsing message", "cid", m.Cid().String(), "to", "receipt", r, m.VMMessage().To)
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid(),
					Error: fmt.Errorf("failed to parse message params: missing to actor code").Error(),
				})
				continue
			}

			// the message applied successfully and we found its actor code, failing to parse here indicates an error.
			method, params, err := util.MethodAndParamsForMessage(m.VMMessage(), toActorCode)
			if err != nil {
				errStr := fmt.Sprintf("failed to parse message: %s, to: %s, receipt: %v, error: %s", m.Cid(), m.VMMessage().To, r, err)
				log.Error(errStr)
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid(),
					Error: errStr,
				})
			}
			pm := &messagemodel.ParsedMessage{
				Height: int64(msgrec.Block.Height),
				Cid:    m.Cid().String(),
				From:   m.VMMessage().From.String(),
				To:     m.VMMessage().To.String(),
				Value:  m.VMMessage().Value.String(),
				Method: method,
				Params: params,
			}
			parsedMessageResults = append(parsedMessageResults, pm)
		}
	}
	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		parsedMessageResults,
	}, report, nil
}

package parsedmessage

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/messages"
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

	var getActorCodeFn func(address address.Address) (cid.Cid, bool)
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
		exeMsgSeen           = make(map[cid.Cid]bool)
		totalGasLimit        int64
		totalUniqGasLimit    int64
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

		// calculate total gas limit of executed messages regardless of duplicates.
		for itr.HasNext() {
			msg, _, _ := itr.Next()
			totalGasLimit += msg.VMMessage().GasLimit
		}

		// reset the iterator to beginning
		itr.Reset()

		for itr.HasNext() {
			m, _, r := itr.Next()
			if exeMsgSeen[m.Cid()] {
				continue
			}
			exeMsgSeen[m.Cid()] = true
			totalUniqGasLimit += m.VMMessage().GasLimit

			toActorCode, found := getActorCodeFn(m.VMMessage().To)
			if !found && r.ExitCode == 0 {
				// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
				// message fails then Lotus will not allow the actor to be created and we are left with an address of an
				// unknown type.
				// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
				// indicates that Lily actor type detection is out of date.
				log.Errorw("parsing message", "error", err, "cid", m.Cid().String(), "receipt", r)
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid(),
					Error: fmt.Errorf("failed to parse message params: missing to actor code").Error(),
				})
			} else {
				method, params, err := util.MethodAndParamsForMessage(m.VMMessage(), toActorCode)
				if err == nil {
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
				} else {
					if r.ExitCode == exitcode.ErrSerialization ||
						r.ExitCode == exitcode.ErrIllegalArgument ||
						r.ExitCode == exitcode.SysErrInvalidMethod ||
						// UsrErrUnsupportedMethod TODO: https://github.com/filecoin-project/go-state-types/pull/44
						r.ExitCode == exitcode.ExitCode(22) {
						// ignore the parse error since the params are probably malformed, as reported by the vm

					} else {
						log.Errorw("parsing message", "error", err, "cid", m.Cid().String(), "receipt", r)
						errorsDetected = append(errorsDetected, &messages.MessageError{
							Cid:   m.Cid(),
							Error: fmt.Errorf("failed to parse message params: %w", err).Error(),
						})
					}
				}
			}
		}
	}
	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		parsedMessageResults,
	}, report, nil
}

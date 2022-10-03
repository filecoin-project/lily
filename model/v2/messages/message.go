package messages

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

func init() {
	v2.RegisterExtractor(&Message{}, ExtractMessages)
}

var _ v2.LilyModel = (*Message)(nil)

type Message struct {
	Height         abi.ChainEpoch
	StateRoot      cid.Cid
	MessageCid     cid.Cid
	ToActorCode    cid.Cid
	From           address.Address
	To             address.Address
	Value          abi.TokenAmount
	GasFeeCap      abi.TokenAmount
	GasPremium     abi.TokenAmount
	SizeBytes      int64
	GasLimit       int64
	Nonce          uint64
	Method         abi.MethodNum
	MessageVersion uint64
	Params         []byte
}

func (t *Message) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(Message{}).Name()),
		Kind:    v2.ModelTsKind,
	}
}

func (t *Message) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func ExtractMessages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]v2.LilyModel, error) {
	grp, _ := errgroup.WithContext(ctx)

	var getActorCodeFn func(address address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCodeFn, err = util.MakeGetActorCodeFunc(ctx, api.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("getting actor code lookup function: %w", err)
		}
		return nil
	})

	var blkMsgRec []*lens.BlockMessageReceipts
	grp.Go(func() error {
		var err error
		blkMsgRec, err = api.TipSetMessageReceipts(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages and receipts: %w", err)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	var (
		out        = make([]v2.LilyModel, 0, len(blkMsgRec))
		exeMsgSeen = make(map[cid.Cid]bool)
	)

	for _, msgrec := range blkMsgRec {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		itr, err := msgrec.Iterator()
		if err != nil {
			return nil, err
		}

		for itr.HasNext() {
			m, _, r := itr.Next()
			if exeMsgSeen[m.Cid()] {
				continue
			}
			exeMsgSeen[m.Cid()] = true

			toActorCode, found := getActorCodeFn(m.VMMessage().To)
			if !found && r.ExitCode == 0 {
				// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
				// message fails then Lotus will not allow the actor to be created and we are left with an address of an
				// unknown type.
				// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
				// indicates that Lily actor type detection is out of date.
				log.Errorw("parsing message", "cid", m.Cid().String(), "receipt", r)
				return nil, fmt.Errorf("failed to parse message params: missing to actor code")
			}
			out = append(out, &Message{
				Height:         current.Height(),
				StateRoot:      current.ParentState(),
				MessageCid:     m.Cid(),
				ToActorCode:    toActorCode,
				From:           m.VMMessage().From,
				To:             m.VMMessage().To,
				Value:          m.VMMessage().Value,
				GasFeeCap:      m.VMMessage().GasFeeCap,
				GasPremium:     m.VMMessage().GasPremium,
				SizeBytes:      int64(m.ChainLength()),
				GasLimit:       m.VMMessage().GasLimit,
				Nonce:          m.VMMessage().Nonce,
				Method:         m.VMMessage().Method,
				MessageVersion: m.VMMessage().Version,
				Params:         m.VMMessage().Params,
			})
		}
	}
	return out, nil
}

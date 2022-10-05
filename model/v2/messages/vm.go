package messages

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("shitidk")

func init() {
	v2.RegisterExtractor(&VMMessage{}, Extract)
}

var _ v2.LilyModel = (*VMMessage)(nil)

type VMMessage struct {
	Height      abi.ChainEpoch
	StateRoot   cid.Cid
	SourceCID   cid.Cid
	ToActorCode cid.Cid
	MessageCID  cid.Cid
	From        address.Address
	To          address.Address
	Value       big.Int
	Method      abi.MethodNum
	ExitCode    exitcode.ExitCode
	GasUsed     int64
	Params      []byte
	Return      []byte
}

func (t *VMMessage) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: t.Version(),
		Type:    t.Type(),
		Kind:    v2.ModelTsKind,
	}
}

func (t *VMMessage) Type() v2.ModelType {
	// eww gross
	return v2.ModelType(reflect.TypeOf(VMMessage{}).Name())
}

func (t *VMMessage) Version() v2.ModelVersion {
	return 1
}

func (t *VMMessage) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func (t *VMMessage) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *VMMessage) ToStorageBlock() (block.Block, error) {
	data, err := t.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (t *VMMessage) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func Extract(ctx context.Context, api tasks.DataSource, current *types.TipSet, executed *types.TipSet) ([]v2.LilyModel, error) {
	// execute in parallel as both operations are slow
	grp, _ := errgroup.WithContext(ctx)
	var mex []*lens.MessageExecution
	grp.Go(func() error {
		var err error
		mex, err = api.MessageExecutions(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages executions for tipset: %w", err)
		}
		return nil
	})

	var getActorCode func(a address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		// TODO make this an api method
		getActorCode, err = util.MakeGetActorCodeFunc(ctx, api.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("failed to make actor code query function: %w", err)
		}
		return nil
	})

	// if either fail, report error and bail
	if err := grp.Wait(); err != nil {
		return nil, err
	}

	out := make([]v2.LilyModel, 0, len(mex))
	for _, parentMsg := range mex {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		// TODO this loop could be parallelized if it becomes a bottleneck.
		// NB: the getActorCode method is the expensive call since it resolves addresses and may load the statetree.
		for _, child := range util.GetChildMessagesOf(parentMsg) {
			// Cid() computes a CID, so only call it once
			childCid := child.Message.Cid()

			toCode, found := getActorCode(child.Message.To)
			if !found && child.Receipt.ExitCode == 0 {
				// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
				// message fails then Lotus will not allow the actor to be created, and we are left with an address of an
				// unknown type.
				// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
				// indicates that Lily actor type detection is out of date.
				log.Errorw("extracting VM message", "source_cid", parentMsg.Cid, "source_receipt", parentMsg.Ret, "child_cid", childCid, "child_receipt", child.Receipt)
				return nil, fmt.Errorf("extracting VM message from source messages %s failed to get to actor code for message: %s to address %s", parentMsg.Cid, childCid, child.Message.To)
			}

			out = append(out, &VMMessage{
				Height:      parentMsg.Height,
				StateRoot:   parentMsg.StateRoot,
				SourceCID:   parentMsg.Cid,
				ToActorCode: toCode,
				MessageCID:  childCid,
				From:        child.Message.From,
				To:          child.Message.To,
				Value:       child.Message.Value,
				Method:      child.Message.Method,
				ExitCode:    child.Receipt.ExitCode,
				GasUsed:     child.Receipt.GasUsed,
				Params:      child.Message.Params,
				Return:      child.Receipt.Return,
			})
		}
	}
	return out, nil
}

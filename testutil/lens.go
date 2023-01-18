package testutil

import (
	"bytes"
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	itestkit "github.com/filecoin-project/lotus/itests/kit"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/lens"
)

func NewAPIWrapper(node *itestkit.TestFullNode) lens.API {
	return &APIWrapper{TestFullNode: node}
}

type APIWrapper struct {
	*itestkit.TestFullNode
	ctx context.Context
}

func (aw *APIWrapper) GetMessageExecutionsForTipSetV2(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecutionV2, error) {
	//TODO implement me
	panic("implement me")
}

func (aw *APIWrapper) BurnFundsFn(ctx context.Context, ts *types.TipSet) (lens.ShouldBurnFn, error) {
	//TODO implement me
	panic("implement me")
}

func (aw *APIWrapper) GetMessageExecutionsForTipSet(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error) {
	//TODO implement me
	panic("implement me")
}

func (aw *APIWrapper) ComputeBaseFee(ctx context.Context, ts *types.TipSet) (abi.TokenAmount, error) {
	//TODO implement me
	panic("implement me")
}

func (aw *APIWrapper) MessagesForTipSetBlocks(ctx context.Context, ts *types.TipSet) ([]*lens.BlockMessages, error) {
	//TODO implement me
	panic("implement me")
}

func (aw *APIWrapper) TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*lens.BlockMessageReceipts, error) {
	//TODO implement me
	panic("implement me")
}

func (aw *APIWrapper) CirculatingSupply(ctx context.Context, key types.TipSetKey) (api.CirculatingSupply, error) {
	return aw.StateVMCirculatingSupplyInternal(ctx, key)
}

func (aw *APIWrapper) ChainGetTipSetAfterHeight(ctx context.Context, epoch abi.ChainEpoch, key types.TipSetKey) (*types.TipSet, error) {
	panic("implement me")
}

func (aw *APIWrapper) Store() adt.Store {
	return aw
}

func (aw *APIWrapper) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return fmt.Errorf("out parameter does not implement CBORUnmarshaler")
	}

	// miss :(
	raw, err := aw.ChainReadObj(ctx, c)
	if err != nil {
		return fmt.Errorf("read obj: %w", err)
	}

	if err := cu.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
		return fmt.Errorf("unmarshal obj: %w", err)
	}

	return nil
}

func (aw *APIWrapper) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return cid.Undef, fmt.Errorf("put is not implemented")
}

func (aw *APIWrapper) Context() context.Context {
	return aw.ctx
}

func (aw *APIWrapper) StateGetReceipt(ctx context.Context, msg cid.Cid, from types.TipSetKey) (*types.MessageReceipt, error) {
	ml, err := aw.StateSearchMsg(ctx, from, msg, api.LookbackNoLimit, true)
	if err != nil {
		return nil, err
	}

	if ml == nil {
		return nil, nil
	}

	return &ml.Receipt, nil
}

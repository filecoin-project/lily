package testutil

import (
	"bytes"
	"context"

	"github.com/filecoin-project/lotus/api"
	apitest "github.com/filecoin-project/lotus/api/test"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
)

type APIOpener struct {
	node apitest.TestNode
}

func NewAPIOpener(node apitest.TestNode) *APIOpener {
	return &APIOpener{node: node}
}

func (o *APIOpener) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	return &APIWrapper{
		TestNode: o.node,
		ctx:      ctx,
	}, lens.APICloser(func() {}), nil
}

type APIWrapper struct {
	apitest.TestNode
	ctx context.Context
}

func (aw *APIWrapper) Store() adt.Store {
	return aw
}

func (aw *APIWrapper) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	return nil, xerrors.Errorf("GetExecutedMessagesForTipset is not implemented")
}

func (aw *APIWrapper) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return xerrors.Errorf("out parameter does not implement CBORUnmarshaler")
	}

	// miss :(
	raw, err := aw.ChainReadObj(ctx, c)
	if err != nil {
		return xerrors.Errorf("read obj: %w", err)
	}

	if err := cu.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
		return xerrors.Errorf("unmarshal obj: %w", err)
	}

	return nil
}

func (aw *APIWrapper) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return cid.Undef, xerrors.Errorf("put is not implemented")
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

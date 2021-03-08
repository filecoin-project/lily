package impl

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/bufbstore"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
)

type Wrapper struct {
	impl.FullNodeAPI
}

func (w *Wrapper) Store() adt.Store {
	bs := w.FullNodeAPI.ChainAPI.Chain.Blockstore()
	cachedStore := bufbstore.NewBufferedBstore(bs)
	cs := cbor.NewCborStore(cachedStore)
	adtStore := adt.WrapStore(context.TODO(), cs)
	return adtStore
}

func (w *Wrapper) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	return util.GetExecutedMessagesForTipset(ctx, w.FullNodeAPI.ChainAPI.Chain, ts, pts)
}

func (w *Wrapper) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	return w, nil, nil
}

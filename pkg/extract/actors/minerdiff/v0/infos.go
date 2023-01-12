package v0

import (
	"bytes"
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*InfoChange)(nil)

type InfoChange struct {
	Info   *typegen.Deferred `cborgen:"info"`
	Change core.ChangeType   `cborgen:"change"`
}

func (i *InfoChange) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := i.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil

}

func (i *InfoChange) ToStorageBlock() (block.Block, error) {
	data, err := i.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.WithCodec(cid.Raw).Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func DecodeInfo(b []byte) (*InfoChange, error) {
	var info InfoChange
	if err := info.UnmarshalCBOR(bytes.NewReader(b)); err != nil {
		return nil, err
	}

	return &info, nil
}

const KindMinerInfo = "miner_info"

func (i *InfoChange) Kind() actors.ActorStateKind {
	return KindMinerInfo
}

var _ actors.ActorStateDiff = (*Info)(nil)

type Info struct{}

func (Info) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMinerInfo, zap.Inline(act), "duration", time.Since(start))
	}()
	return InfoDiff(ctx, api, act)
}

type DiffInfoAPI interface {
	Store() adt.Store
	ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error)
	MinerLoad(store adt.Store, act *types.Actor) (miner.State, error)
	Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
}

// separate method for testing purposes

func InfoDiff(ctx context.Context, api DiffInfoAPI, act *actors.ActorChange) (*InfoChange, error) {
	// was removed, no new info
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
	}

	currentMiner, err := api.MinerLoad(api.Store(), act.Current)
	if err != nil {
		return nil, err
	}
	infoBytes, err := api.ChainReadObj(ctx, currentMiner.InfoCid())
	if err != nil {
		return nil, err
	}
	// was added, info is new
	if act.Type == core.ChangeTypeAdd {
		return &InfoChange{
			Info:   &typegen.Deferred{Raw: infoBytes},
			Change: core.ChangeTypeAdd,
		}, nil
	}

	executedMiner, err := api.MinerLoad(api.Store(), act.Executed)
	if err != nil {
		return nil, err
	}
	// wasn't modified
	if executedMiner.InfoCid().Equals(currentMiner.InfoCid()) {
		return nil, nil
	}
	return &InfoChange{
		Info:   &typegen.Deferred{Raw: infoBytes},
		Change: core.ChangeTypeModify,
	}, nil
}
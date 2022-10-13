package tipset

import (
	"bytes"
	"context"
	"reflect"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

func init() {
	v2.RegisterExtractor(&TipSetState{}, Extract)
}

type TipSetState struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	CIDs      []cid.Cid

	ParentHeight    abi.ChainEpoch
	ParentStateRoot cid.Cid
	ParentCIDs      []cid.Cid
}

func (m *TipSetState) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(TipSetState{}).Name()),
		Kind:    v2.ModelTsKind,
	}
}
func (t *TipSetState) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func (t *TipSetState) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *TipSetState) ToStorageBlock() (block.Block, error) {
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

func (t *TipSetState) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func Extract(ctx context.Context, api tasks.DataSource, current *types.TipSet, executed *types.TipSet) ([]v2.LilyModel, error) {
	return []v2.LilyModel{
		&TipSetState{
			Height:          current.Height(),
			StateRoot:       current.ParentState(),
			CIDs:            current.Cids(),
			ParentHeight:    executed.Height(),
			ParentCIDs:      executed.Cids(),
			ParentStateRoot: executed.ParentState(),
		},
	}, nil
}

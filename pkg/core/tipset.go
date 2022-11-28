package core

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/tasks"
)

type TipSetState struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	CIDs      []cid.Cid
	Blocks    []*types.BlockHeader

	ParentHeight    abi.ChainEpoch
	ParentStateRoot cid.Cid
	ParentCIDs      []cid.Cid
}

func ExtractTipSetState(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*TipSetState, error) {
	return &TipSetState{
		Height:          current.Height(),
		StateRoot:       current.ParentState(),
		CIDs:            current.Cids(),
		Blocks:          current.Blocks(),
		ParentHeight:    executed.Height(),
		ParentCIDs:      executed.Cids(),
		ParentStateRoot: executed.ParentState(),
	}, nil
}

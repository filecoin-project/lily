package consensus

import (
	"context"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/chain"
	"github.com/filecoin-project/lotus/chain/types"
)

func init() {
	model.RegisterTipSetModelExtractor(&chain.ChainConsensus{}, ChainConsensusExtractor{})
}

var _ model.TipSetStateExtractor = (*ChainConsensusExtractor)(nil)

type ChainConsensusExtractor struct{}

func (ChainConsensusExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	pl := make(chain.ChainConsensusList, current.Height()-previous.Height())
	idx := 0
	// walk from head to the previous
	for epoch := current.Height(); epoch > previous.Height(); epoch-- {
		if current.Height() == epoch {
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: current.ParentState().String(),
				ParentTipSet:    current.Parents().String(),
				TipSet:          current.Key().String(),
			}
		} else {
			// null round no tipset
			pl[idx] = &chain.ChainConsensus{
				Height:          int64(epoch),
				ParentStateRoot: current.ParentState().String(),
				ParentTipSet:    current.Parents().String(),
				TipSet:          "",
			}
		}
		idx += 1
	}
	return pl, nil
}

func (ChainConsensusExtractor) Name() string {
	return "chain_consensus"
}

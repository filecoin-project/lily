package tipset

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model/chain"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/tipset"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("transform/tipset")

type ConsensusTransform struct {
	meta v2.ModelMeta
}

func NewConsensusTransform() *ConsensusTransform {
	info := tipset.TipSetState{}
	return &ConsensusTransform{meta: info.Meta()}
}

func (s *ConsensusTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(chain.ChainConsensusList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				ts := modeldata.(*tipset.TipSetState)
				currentHeight := ts.Height
				parentHeight := ts.ParentHeight
				for epoch := currentHeight; epoch > parentHeight; epoch-- {
					if currentHeight == epoch {
						sqlModels = append(sqlModels, &chain.ChainConsensus{
							Height:          int64(epoch),
							ParentStateRoot: ts.StateRoot.String(),
							ParentTipSet:    types.NewTipSetKey(ts.ParentCIDs...).String(),
							TipSet:          types.NewTipSetKey(ts.CIDs...).String(),
						})
					} else {
						// null round no tipset
						sqlModels = append(sqlModels, &chain.ChainConsensus{
							Height:          int64(epoch),
							ParentStateRoot: ts.ParentStateRoot.String(),
							ParentTipSet:    types.NewTipSetKey(ts.ParentCIDs...).String(),
							TipSet:          "",
						})
					}
				}
				if ts.Height == 0 {
					sqlModels = append(sqlModels, &chain.ChainConsensus{
						Height:          int64(ts.Height),
						ParentStateRoot: ts.StateRoot.String(),
						ParentTipSet:    types.NewTipSetKey(ts.ParentCIDs...).String(),
						TipSet:          types.NewTipSetKey(ts.CIDs...).String(),
					})
				}
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *ConsensusTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *ConsensusTransform) Name() string {
	info := ConsensusTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *ConsensusTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

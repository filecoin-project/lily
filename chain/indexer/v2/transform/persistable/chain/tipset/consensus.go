package tipset

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	chain2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/chain"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/tipset"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

var log = logging.Logger("transform/tipset")

type ConsensusTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewConsensusTransform(taskName string) *ConsensusTransform {
	info := tipset.TipSetState{}
	return &ConsensusTransform{meta: info.Meta(), taskName: taskName}
}

func (s *ConsensusTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain2.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			rp := make(visormodel.ProcessingReportList, 0)
			sqlModels := make(chain.ChainConsensusList, 0, len(res.Models))
			for _, modeldata := range res.Models {
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
				for epoch := currentHeight; epoch > parentHeight; epoch-- {
					if currentHeight == epoch {
						rp = append(rp, &visormodel.ProcessingReport{
							Height:    int64(epoch),
							StateRoot: ts.ParentStateRoot.String(),
						})
					} else {
						// null round no tipset
						rp = append(rp, &visormodel.ProcessingReport{
							Height:            int64(epoch),
							StateRoot:         ts.ParentStateRoot.String(),
							StatusInformation: visormodel.ProcessingStatusInformationNullRound,
						})
					}
				}
			}
			if len(sqlModels) > 0 {
				data = append(data, sqlModels)
				data = append(data, rp)
			}
			out <- &persistable.Result{Model: data}
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

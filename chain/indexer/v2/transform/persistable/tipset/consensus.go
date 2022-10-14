package tipset

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/chain"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/tipset"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

var log = logging.Logger("transform/tipset")

type ConsensusTransform struct {
	meta v2.ModelMeta
}

func NewConsensusTransform() *ConsensusTransform {
	info := tipset.TipSetState{}
	return &ConsensusTransform{meta: info.Meta()}
}

func (s *ConsensusTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		start := time.Now()
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(chain.ChainConsensusList, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
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
			report := &visormodel.ProcessingReport{
				Height:            int64(res.Current().Height()),
				StateRoot:         res.Current().ParentState().String(),
				Reporter:          "TODO",
				Task:              s.Name(),
				StartedAt:         start,
				CompletedAt:       time.Now(),
				Status:            visormodel.ProcessingStatusOK,
				StatusInformation: "",
				ErrorsDetected:    nil,
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: model.PersistableList{sqlModels, report}}
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

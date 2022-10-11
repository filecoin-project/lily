package message

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"reflect"

	"github.com/filecoin-project/lotus/build"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
	"github.com/filecoin-project/lily/tasks"
)

type GasEconomyTransform struct {
	meta v2.ModelMeta
}

func NewGasEconomyTransform() *GasEconomyTransform {
	info := messages.BlockMessage{}
	return &GasEconomyTransform{meta: info.Meta()}
}

func (g *GasEconomyTransform) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", g.Name())
	var (
		seenMsgs          = cid.NewSet()
		totalGasLimit     int64
		totalUniqGasLimit int64
	)
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Debugw("received data", "count", len(res.State().Data))
			// add up total and unique gas
			for _, modeldata := range res.State().Data {
				m := modeldata.(*messages.BlockMessage)
				totalGasLimit += m.GasLimit
				if seenMsgs.Visit(m.MessageCid) {
					totalUniqGasLimit += m.GasLimit
				}
			}

			// TODO pull this out into an extractor to remove the need of running a lily node with API to compute gas base feed.
			currentBaseFee, err := api.ComputeBaseFee(ctx, res.Current())
			if err != nil {
				return err
			}
			baseFeeRat := new(big.Rat).SetFrac(currentBaseFee.Int, new(big.Int).SetUint64(build.FilecoinPrecision))
			baseFee, _ := baseFeeRat.Float64()

			baseFeeChange := new(big.Rat).SetFrac(currentBaseFee.Int, res.Current().Blocks()[0].ParentBaseFee.Int)
			baseFeeChangeF, _ := baseFeeChange.Float64()

			sqlModel := &messagemodel.MessageGasEconomy{
				Height:              int64(res.Current().Height()),
				StateRoot:           res.Current().ParentState().String(),
				GasLimitTotal:       totalGasLimit,
				GasLimitUniqueTotal: totalUniqGasLimit,
				BaseFee:             baseFee,
				BaseFeeChangeLog:    math.Log(baseFeeChangeF) / math.Log(1.125),
				GasFillRatio:        float64(totalGasLimit) / float64(len(res.Current().Blocks())*build.BlockGasTarget),
				GasCapacityRatio:    float64(totalUniqGasLimit) / float64(len(res.Current().Blocks())*build.BlockGasTarget),
				GasWasteRatio:       float64(totalGasLimit-totalUniqGasLimit) / float64(len(res.Current().Blocks())*build.BlockGasTarget),
			}
			out <- &persistable.Result{Model: sqlModel}
		}
	}
	return nil
}

func (g *GasEconomyTransform) Name() string {
	info := GasEconomyTransform{}
	return reflect.TypeOf(info).Name()
}

func (g *GasEconomyTransform) ModelType() v2.ModelMeta {
	return g.meta
}

func (g *GasEconomyTransform) Matcher() string {
	return fmt.Sprintf("^%s$", g.meta.String())
}

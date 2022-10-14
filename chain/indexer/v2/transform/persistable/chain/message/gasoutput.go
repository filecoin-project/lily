package message

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain"
	"github.com/filecoin-project/lily/model"
	derivedmodel "github.com/filecoin-project/lily/model/derived"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/messages"
)

type GasOutputTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewGasOutputTransform(taskName string) *GasOutputTransform {
	info := messages.ExecutedMessage{}
	return &GasOutputTransform{meta: info.Meta(), taskName: taskName}
}

func (g *GasOutputTransform) Run(ctx context.Context, reporter string, in chan *extract.TipSetStateResult, out chan transform.Result) error {
	log.Debugf("run %s", g.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := chain.ToProcessingReport(g.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Models))
			sqlModels := make(derivedmodel.GasOutputsList, 0, len(res.Models))
			for _, modeldata := range res.Models {
				m := modeldata.(*messages.ExecutedMessage)

				actorName := builtin.ActorNameByCode(m.ToActorCode)
				sqlModels = append(sqlModels, &derivedmodel.GasOutputs{
					Height:             int64(m.Height),
					Cid:                m.MessageCid.String(),
					StateRoot:          m.StateRoot.String(),
					From:               m.From.String(),
					To:                 m.To.String(),
					Value:              m.Value.String(),
					GasFeeCap:          m.GasFeeCap.String(),
					GasPremium:         m.GasPremium.String(),
					GasLimit:           m.GasLimit,
					SizeBytes:          int(m.SizeBytes),
					Nonce:              m.Nonce,
					Method:             uint64(m.Method),
					ActorName:          actorName,
					ActorFamily:        builtin.ActorFamily(actorName),
					ExitCode:           int64(m.ExitCode),
					GasUsed:            m.GasUsed,
					ParentBaseFee:      m.ParentBaseFee.String(),
					BaseFeeBurn:        m.BaseFeeBurn.String(),
					OverEstimationBurn: m.OverEstimationBurn.String(),
					MinerPenalty:       m.MinerPenalty.String(),
					MinerTip:           m.MinerTip.String(),
					Refund:             m.Refund.String(),
					GasRefund:          m.GasRefund,
					GasBurned:          m.GasBurned,
				})
			}
			if len(sqlModels) > 0 {
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
		}
	}
	return nil
}

func (g *GasOutputTransform) Name() string {
	info := GasOutputTransform{}
	return reflect.TypeOf(info).Name()
}

func (g *GasOutputTransform) ModelType() v2.ModelMeta {
	return g.meta
}

func (g *GasOutputTransform) Matcher() string {
	return fmt.Sprintf("^%s$", g.meta.String())
}

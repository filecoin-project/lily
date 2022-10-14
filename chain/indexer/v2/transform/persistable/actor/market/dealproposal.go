package market

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/text/runes"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/market"
)

var log = logging.Logger("transform/dealproposal")

type DealProposalTransformer struct {
	meta     v2.ModelMeta
	taskName string
}

func NewDealProposalTransformer(taskName string) *DealProposalTransformer {
	info := market.DealProposal{}
	return &DealProposalTransformer{meta: info.Meta(), taskName: taskName}
}

func (d *DealProposalTransformer) ModelType() v2.ModelMeta {
	return d.meta
}

func (d *DealProposalTransformer) Name() string {
	info := DealProposalTransformer{}
	return reflect.TypeOf(info).Name()
}

func (d *DealProposalTransformer) Matcher() string {
	return fmt.Sprintf("^%s$", d.meta.String())
}

func (d *DealProposalTransformer) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", d.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(d.taskName, reporter, res)
			data := model.PersistableList{report}
			sqlModels := make(marketmodel.MarketDealProposals, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				se := modeldata.(*market.DealProposal)
				var base64Label string
				if se.Label.IsString {
					base64Label = base64.StdEncoding.EncodeToString([]byte(SanitizeLabel(string(se.Label.Label))))

				} else {
					base64Label = base64.StdEncoding.EncodeToString(se.Label.Label)

				}

				sqlModels = append(sqlModels, &marketmodel.MarketDealProposal{
					Height:               int64(se.Height),
					DealID:               uint64(se.DealID),
					StateRoot:            se.StateRoot.String(),
					PaddedPieceSize:      uint64(se.PieceSize),
					UnpaddedPieceSize:    uint64(se.PieceSize.Unpadded()),
					StartEpoch:           int64(se.StartEpoch),
					EndEpoch:             int64(se.EndEpoch),
					ClientID:             se.Client.String(),
					ProviderID:           se.Provider.String(),
					ClientCollateral:     se.ClientCollateral.String(),
					ProviderCollateral:   se.ProviderCollateral.String(),
					StoragePricePerEpoch: se.StoragePricePerEpoch.String(),
					PieceCID:             se.PieceCID.String(),
					IsVerified:           se.VerifiedDeal,
					Label:                base64Label,
					IsString:             se.Label.IsString,
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

func SanitizeLabel(s string) string {
	if s == "" {
		return s
	}
	s = strings.Replace(s, "\000", "", -1)
	if utf8.ValidString(s) {
		return s
	}

	tr := runes.ReplaceIllFormed()
	return tr.String(s)
}

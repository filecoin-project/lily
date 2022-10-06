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

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/market"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("transform/dealproposal")

type DealProposalTransformer struct {
	meta v2.ModelMeta
}

func NewDealProposalTransformer() *DealProposalTransformer {
	info := market.DealProposal{}
	return &DealProposalTransformer{meta: info.Meta()}
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

func (d *DealProposalTransformer) Run(ctx context.Context, api tasks.DataSource, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", d.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(marketmodel.MarketDealProposals, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
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
				out <- &persistable.Result{Model: sqlModels}
			}
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

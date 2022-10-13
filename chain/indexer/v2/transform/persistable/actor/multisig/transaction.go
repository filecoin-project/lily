package multisig

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	multisigmodel "github.com/filecoin-project/lily/model/actors/multisig"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/multisig"
)

var log = logging.Logger("transform/multisig")

type TransactionTransform struct {
	meta v2.ModelMeta
}

func NewTransactionTransform() *TransactionTransform {
	info := multisig.MultisigTransaction{}
	return &TransactionTransform{meta: info.Meta()}
}

func (s *TransactionTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(multisigmodel.MultisigTransactionList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				tx := modeldata.(*multisig.MultisigTransaction)
				if tx.Event == multisig.Added || tx.Event == multisig.Modified {
					approved := make([]string, len(tx.Approved))
					for i, addr := range tx.Approved {
						approved[i] = addr.String()
					}
					sqlModels = append(sqlModels, &multisigmodel.MultisigTransaction{
						MultisigID:    tx.Multisig.String(),
						StateRoot:     tx.StateRoot.String(),
						Height:        int64(tx.Height),
						TransactionID: tx.TransactionID,
						To:            tx.To.String(),
						Value:         tx.Value.String(),
						Method:        uint64(tx.Method),
						Params:        tx.Params,
						Approved:      approved,
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

func (s *TransactionTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *TransactionTransform) Name() string {
	info := TransactionTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *TransactionTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

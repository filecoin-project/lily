package multisig

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MultisigTaskResult struct {
	TransactionModel MultisigTransactionList
}

func (mtr *MultisigTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(mtr.TransactionModel) > 0 {
		return mtr.TransactionModel.Persist(ctx, s, version)
	}
	return nil
}

type MultisigTaskResultList []*MultisigTaskResult

func (ml MultisigTaskResultList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	for _, res := range ml {
		if err := res.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	return nil
}

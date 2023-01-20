package v9

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	verifregdiff "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v9"
)

type Claims struct{}

func (Claims) Extract(ctx context.Context, current, executed *types.TipSet, changes *verifregdiff.StateDiffResult) (model.Persistable, error) {
	panic("TODO")
}

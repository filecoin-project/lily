package v9

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	verifregdiff "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v2"
)

type Claims struct{}

func (Claims) Transform(ctx context.Context, current, executed *types.TipSet, changes *verifregdiff.StateDiffResult) (model.Persistable, error) {
	panic("TODO")
}

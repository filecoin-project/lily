package cborable

import (
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	v2 "github.com/filecoin-project/lily/model/v2"
)

type Result struct {
	TipSet *types.TipSet
	Model  []v2.LilyModel
}

type CborablResult struct {
	TipSet *types.TipSet
	Model  []v2.LilyModel
}

func (r *Result) Kind() transform.Kind {
	return "cborable"
}

func (r *Result) Data() interface{} {
	return CborablResult{
		TipSet: r.TipSet,
		Model:  r.Model,
	}
}

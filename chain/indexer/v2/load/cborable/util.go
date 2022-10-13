package cborable

import (
	"strings"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	v2 "github.com/filecoin-project/lily/model/v2"
)

func ParseTipSetString(ts string) ([]cid.Cid, error) {
	strs := strings.Split(ts, ",")

	var cids []cid.Cid
	for _, s := range strs {
		c, err := cid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	return cids, nil
}

type ModelKeyer struct {
	M v2.ModelMeta
}

func (m ModelKeyer) Key() string {
	return m.M.String()
}

type TipsetKeyer struct {
	T types.TipSetKey
}

func (m TipsetKeyer) Key() string {
	return m.T.String()
}

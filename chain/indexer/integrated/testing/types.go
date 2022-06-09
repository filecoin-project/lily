package testing

import (
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

var dummyCid cid.Cid

func init() {
	dummyCid, _ = cid.Parse("bafkqaaa")
}

func MustFakeTipSet(t *testing.T, height int64) *types.TipSet {
	minerAddr, err := address.NewFromString("t00")
	require.NoError(t, err)

	ts, err := types.NewTipSet([]*types.BlockHeader{{
		Miner:                 minerAddr,
		Height:                abi.ChainEpoch(height),
		ParentStateRoot:       dummyCid,
		Messages:              dummyCid,
		ParentMessageReceipts: dummyCid,
		BlockSig:              &crypto.Signature{Type: crypto.SigTypeBLS},
		BLSAggregate:          &crypto.Signature{Type: crypto.SigTypeBLS},
		Timestamp:             1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

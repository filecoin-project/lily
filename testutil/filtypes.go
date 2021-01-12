package testutil

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/ipfs/go-cid"
)

func FakeTipset(t testing.TB) *types.TipSet {
	bh := FakeBlockHeader(t, 1, RandomCid())
	ts, err := types.NewTipSet([]*types.BlockHeader{bh})
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func FakeBlockHeader(t testing.TB, height int64, stateRoot cid.Cid) *types.BlockHeader {
	return &types.BlockHeader{
		Miner:                 tutils.NewIDAddr(t, 123),
		Height:                abi.ChainEpoch(height),
		ParentStateRoot:       stateRoot,
		Messages:              RandomCid(),
		ParentMessageReceipts: RandomCid(),
		BlockSig:              &crypto.Signature{Type: crypto.SigTypeBLS},
		BLSAggregate:          &crypto.Signature{Type: crypto.SigTypeBLS},
		Timestamp:             0,
	}
}

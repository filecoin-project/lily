package testutil

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/lens"
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

type BlockHeaderList []*types.BlockHeader

func (b BlockHeaderList) Cids() []string {
	var cids []string
	for _, bh := range b {
		cids = append(cids, bh.Cid().String())
	}
	return cids
}

func (b BlockHeaderList) Rounds() []uint64 {
	var rounds []uint64
	for _, bh := range b {
		for _, ent := range bh.BeaconEntries {
			rounds = append(rounds, ent.Round)
		}
	}

	return rounds
}

// CollectBlockHeaders walks the chain to collect blocks that should be indexed
func CollectBlockHeaders(n lens.API, ts *types.TipSet) (BlockHeaderList, error) {
	blocks := ts.Blocks()

	for _, bh := range ts.Blocks() {
		if bh.Height < 2 {
			continue
		}

		parent, err := n.ChainGetTipSet(context.TODO(), types.NewTipSetKey(bh.Parents...))
		if err != nil {
			return nil, err
		}

		pblocks, err := CollectBlockHeaders(n, parent)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, pblocks...)

	}
	return blocks, nil
}

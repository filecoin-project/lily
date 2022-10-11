package main

import (
	"fmt"
	"os"

	"github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/model/v2/actors/init_"
	"github.com/filecoin-project/lily/model/v2/actors/market"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
	"github.com/filecoin-project/lily/model/v2/actors/multisig"
	"github.com/filecoin-project/lily/model/v2/actors/power"
	"github.com/filecoin-project/lily/model/v2/actors/raw"
	"github.com/filecoin-project/lily/model/v2/actors/reward"
	"github.com/filecoin-project/lily/model/v2/actors/verifreg"
	"github.com/filecoin-project/lily/model/v2/block"
	"github.com/filecoin-project/lily/model/v2/messages"
)

func main() {
	err := typegen.WriteTupleEncodersToFile("./model/v2/messages/cbor_gen.go", "messages",
		messages.VMMessage{},
		messages.ExecutedMessage{},
		messages.BlockMessage{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/raw/cbor_gen.go", "raw",
		raw.ActorState{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/miner/cbor_gen.go", "miner",
		miner.SectorEvent{},
		miner.PreCommitEvent{},
		miner.SectorPreCommitInfo{},
		miner.SectorPreCommitOnChainInfo{},
		miner.MinerInfo{},
		miner.WorkerKeyChange{},
		miner.LockedFunds{},
		miner.DeadlineInfo{},
		miner.FeeDebt{},
		miner.PostSectorMessage{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/block/cbor_gen.go", "block",
		block.BlockHeader{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/market/cbor_gen.go", "market",
		market.DealProposal{},
		market.DealLabel{},
		market.DealState{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/init_/cbor_gen.go", "init_",
		init_.AddressState{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/multisig/cbor_gen.go", "multisig",
		multisig.MultisigTransaction{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/power/cbor_gen.go", "power",
		power.ChainPower{},
		power.ClaimedPower{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/reward/cbor_gen.go", "reward",
		reward.ChainReward{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/verifreg/cbor_gen.go", "verifreg",
		verifreg.VerifiedClient{},
		verifreg.Verifier{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

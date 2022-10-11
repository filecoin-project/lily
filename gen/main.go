package main

import (
	"fmt"
	"os"

	"github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/model/v2/actors/init_"
	"github.com/filecoin-project/lily/model/v2/actors/market"
	"github.com/filecoin-project/lily/model/v2/actors/miner/precommitevent"
	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorevent"
	"github.com/filecoin-project/lily/model/v2/actors/raw"
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
	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/miner/sectorevent/cbor_gen.go", "sectorevent",
		sectorevent.SectorEvent{},
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
	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/miner/precommitevent/cbor_gen.go", "precommitevent",
		precommitevent.PreCommitEvent{},
		precommitevent.SectorPreCommitInfo{},
		precommitevent.SectorPreCommitOnChainInfo{},
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

}

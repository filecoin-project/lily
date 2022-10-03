package main

import (
	"fmt"
	"os"

	"github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/model/v2/actors/miner/precommitevent"
	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorevent"
	"github.com/filecoin-project/lily/model/v2/block"
	"github.com/filecoin-project/lily/model/v2/messages"
)

func main() {
	err := typegen.WriteTupleEncodersToFile("./model/v2/messages/cbor_gen.go", "messages",
		messages.VMMessage{},
		messages.Message{},
		messages.Receipt{},
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

}

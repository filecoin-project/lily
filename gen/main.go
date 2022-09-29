package main

import (
	"fmt"
	"os"

	"github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorevent"
	"github.com/filecoin-project/lily/model/v2/actors/miner/sectorinfo"
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
	err = typegen.WriteTupleEncodersToFile("./model/v2/actors/miner/sectorinfo/cbor_gen.go", "sectorinfo",
		sectorinfo.SectorInfo{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

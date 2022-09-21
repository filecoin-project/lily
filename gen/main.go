package main

import (
	"fmt"
	"os"

	"github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/model/v2/messages"
)

func main() {
	err := typegen.WriteTupleEncodersToFile("./model/v2/messages/cbor_gen.go", "messages",
		messages.VMMessage{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

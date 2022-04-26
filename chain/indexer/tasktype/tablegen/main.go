package main

import (
	"github.com/filecoin-project/lily/chain/indexer/tasktype/tablegen/generator"
)

func main() {
	if err := generator.Gen(); err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"

	"github.com/filecoin-project/lily/chain/actors/agen/generator"
)

func main() {
	if err := generator.Gen(); err != nil {
		fmt.Println(err)
	}
}

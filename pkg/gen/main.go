package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
)

const minerDiffPath = "pkg/extract/actors/minerdiff/cbor_gen.go"
const minerDiffPkg = "minerdiff"

const minerTransformPath = "pkg/transform/cbor/miner/cbor_gen.go"
const minerTransformPkg = "miner"

const actorTransformPath = "pkg/transform/cbor/cbor_gen.go"
const actorTransformPkg = "cbor"

func main() {
	if err := cbg.WriteMapEncodersToFile(minerDiffPath, minerDiffPkg,
		v0.SectorStatusChange{},
		v0.PreCommitChange{},
		v0.SectorChange{},
		v0.FundsChange{},
		v0.DebtChange{},
		v0.InfoChange{},
		v0.StateChange{},
	); err != nil {
		panic(err)
	}

	/*
		if err := cbg.WriteMapEncodersToFile(minerTransformPath, minerTransformPkg,
			v9.StateChange{},
		); err != nil {
			panic(err)
		}

		if err := cbg.WriteMapEncodersToFile(actorTransformPath, actorTransformPkg,
			cbor.ActorIPLDContainer{},
		); err != nil {
			panic(err)
		}

	*/

}

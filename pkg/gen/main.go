package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/miner"
)

const minerDiffPath = "pkg/extract/actors/minerdiff/cbor_gen.go"
const minerDiffPkg = "minerdiff"

const minerTransformPath = "pkg/transform/cbor/miner/cbor_gen.go"
const minerTransformPkg = "miner"

const actorTransformPath = "pkg/transform/cbor/cbor_gen.go"
const actorTransformPkg = "cbor"

func main() {
	if err := cbg.WriteMapEncodersToFile(minerDiffPath, minerDiffPkg,
		minerdiff.SectorStatusChange{},
		minerdiff.PreCommitChange{},
		minerdiff.SectorChange{},
		minerdiff.FundsChange{},
		minerdiff.DebtChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(minerTransformPath, minerTransformPkg,
		miner.StateChange{},
		miner.Info{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(actorTransformPath, actorTransformPkg,
		cbor.ActorIPLDContainer{},
	); err != nil {
		panic(err)
	}

}

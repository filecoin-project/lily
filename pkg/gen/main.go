package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
	initv0 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v0"
	minerv0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
	verifV0 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
	verifV9 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v9"
)

const actorDiffPath = "pkg/extract/actors/actordiff/cbor_gen.go"
const actorDiffPkg = "actordiff"

const minerDiffPath = "pkg/extract/actors/minerdiff/v0/cbor_gen.go"
const minerDiffPkg = "v0"

const initDiffPath = "pkg/extract/actors/initdiff/v0/cbor_gen.go"
const initDiffPkg = "v0"

const verifDiffPathV0 = "pkg/extract/actors/verifregdiff/v0/cbor_gen.go"
const verifDiffPkgV0 = "v0"

const verifDiffPathV9 = "pkg/extract/actors/verifregdiff/v9/cbor_gen.go"
const verifDiffPkgV9 = "v9"

func main() {
	if err := cbg.WriteMapEncodersToFile(actorDiffPath, actorDiffPkg,
		actordiff.ActorChange{},
		actordiff.StateChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(minerDiffPath, minerDiffPkg,
		minerv0.SectorStatusChange{},
		minerv0.PreCommitChange{},
		minerv0.SectorChange{},
		minerv0.FundsChange{},
		minerv0.DebtChange{},
		minerv0.InfoChange{},
		minerv0.StateChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(initDiffPath, initDiffPkg,
		initv0.AddressChange{},
		initv0.StateChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(verifDiffPathV0, verifDiffPkgV0,
		verifV0.StateChange{},
		verifV0.ClientsChange{},
		verifV0.VerifiersChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(verifDiffPathV9, verifDiffPkgV9,
		verifV9.StateChange{},
		verifV9.ClaimsChange{},
		verifV9.AllocationsChange{},
	); err != nil {
		panic(err)
	}

}

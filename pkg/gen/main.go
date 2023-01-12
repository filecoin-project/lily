package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
	initv0 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v0"
	marketV0 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v0"
	minerV0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
	powerV0 "github.com/filecoin-project/lily/pkg/extract/actors/powerdiff/v0"
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

const marketDiffPath = "pkg/extract/actors/marketdiff/v0/cbor_gen.go"
const marketDiffPkg = "v0"

const powerDiffPath = "pkg/extract/actors/powerdiff/v0/cbor_gen.go"
const powerDiffPkg = "v0"

func main() {
	if err := cbg.WriteMapEncodersToFile(actorDiffPath, actorDiffPkg,
		actordiff.ActorChange{},
		actordiff.StateChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(minerDiffPath, minerDiffPkg,
		minerV0.SectorStatusChange{},
		minerV0.PreCommitChange{},
		minerV0.SectorChange{},
		minerV0.FundsChange{},
		minerV0.DebtChange{},
		minerV0.InfoChange{},
		minerV0.StateChange{},
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

	if err := cbg.WriteMapEncodersToFile(marketDiffPath, marketDiffPkg,
		marketV0.StateChange{},
		marketV0.ProposalChange{},
		marketV0.DealChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(powerDiffPath, powerDiffPkg,
		powerV0.StateChange{},
		powerV0.ClaimsChange{},
	); err != nil {
		panic(err)
	}
}

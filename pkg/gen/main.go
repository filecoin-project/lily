package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"

	datacapV9 "github.com/filecoin-project/lily/pkg/extract/actors/datacapdiff/v1"
	initv0 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v1"
	marketV0 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v1"
	minerV0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"
	powerV0 "github.com/filecoin-project/lily/pkg/extract/actors/powerdiff/v1"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
	verifV0 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v1"
	verifV9 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v2"
	"github.com/filecoin-project/lily/pkg/extract/chain"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors"
	"github.com/filecoin-project/lily/pkg/transform/cbor/messages"
)

const actorDiffPath = "pkg/extract/actors/rawdiff/cbor_gen.go"
const actorDiffPkg = "rawdiff"

const datacapDiffPath = "pkg/extract/actors/datacapdiff/v1/cbor_gen.go"
const datacapDiffPkg = "v1"

const minerDiffPath = "pkg/extract/actors/minerdiff/v1/cbor_gen.go"
const minerDiffPkg = "v1"

const initDiffPath = "pkg/extract/actors/initdiff/v1/cbor_gen.go"
const initDiffPkg = "v1"

const verifDiffPathV0 = "pkg/extract/actors/verifregdiff/v1/cbor_gen.go"
const verifDiffPkgV0 = "v1"

const verifDiffPathV9 = "pkg/extract/actors/verifregdiff/v2/cbor_gen.go"
const verifDiffPkgV9 = "v2"

const marketDiffPath = "pkg/extract/actors/marketdiff/v1/cbor_gen.go"
const marketDiffPkg = "v1"

const powerDiffPath = "pkg/extract/actors/powerdiff/v1/cbor_gen.go"
const powerDiffPkg = "v1"

const IPLDActorContainerPath = "pkg/transform/cbor/actors/cbor_gen.go"
const IPLDActorContainerPkg = "actors"

const MessageStatePath = "pkg/extract/chain/cbor_gen.go"
const MessageStatePkg = "chain"

const MessageContainerPath = "pkg/transform/cbor/messages/cbor_gen.go"
const MessageContainerPkg = "messages"

const RootStatePath = "pkg/transform/cbor/cbor_gen.go"
const RootStatePkg = "cbor"

func main() {
	if err := cbg.WriteMapEncodersToFile(datacapDiffPath, datacapDiffPkg,
		datacapV9.AllowanceChange{},
		datacapV9.BalanceChange{},
		datacapV9.StateChange{},
	); err != nil {
		panic(err)
	}
	if err := cbg.WriteMapEncodersToFile(actorDiffPath, actorDiffPkg,
		rawdiff.ActorChange{},
		rawdiff.StateChange{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(minerDiffPath, minerDiffPkg,
		minerV0.SectorStatusChange{},
		minerV0.PreCommitChange{},
		minerV0.SectorChange{},
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

	if err := cbg.WriteMapEncodersToFile(IPLDActorContainerPath, IPLDActorContainerPkg,
		actors.ActorStateChangesIPLD{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(MessageStatePath, MessageStatePkg,
		chain.ChainMessageReceipt{},
		chain.ImplicitMessageReceipt{},
		chain.MessageGasOutputs{},
		chain.ActorError{},
		chain.VmMessage{},
		chain.VmMessageGasTrace{},
		chain.Loc{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(MessageContainerPath, MessageContainerPkg,
		messages.FullBlockIPLDContainer{},
		messages.ChainMessageIPLDContainer{},
		messages.SignedChainMessageIPLDContainer{},
		messages.ImplicitMessageIPLDContainer{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile(RootStatePath, RootStatePkg,
		cbor.RootStateIPLD{},
		cbor.StateExtractionIPLD{},
	); err != nil {
		panic(err)
	}
}

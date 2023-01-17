package builtin

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"

	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"

	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"

	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"

	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"

	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/proof"

	"github.com/filecoin-project/lotus/chain/actors"

	smoothingtypes "github.com/filecoin-project/go-state-types/builtin/v8/util/smoothing"
)

const (
	EpochDurationSeconds = builtin.EpochDurationSeconds
	EpochsInDay          = builtin.EpochsInDay
	SecondsInDay         = builtin.SecondsInDay
)

const (
	MethodSend        = builtin.MethodSend
	MethodConstructor = builtin.MethodConstructor
)

// These are all just type aliases across actor versions. In the future, that might change
// and we might need to do something fancier.
type SectorInfo = proof.SectorInfo
type ExtendedSectorInfo = proof.ExtendedSectorInfo
type PoStProof = proof.PoStProof
type FilterEstimate = smoothingtypes.FilterEstimate

func ActorNameByCode(c cid.Cid) string {
	if name, version, ok := actors.GetActorMetaByCode(c); ok {
		return fmt.Sprintf("fil/%d/%s", version, name)
	}

	switch {

	case builtin0.IsBuiltinActor(c):
		return builtin0.ActorNameByCode(c)

	case builtin2.IsBuiltinActor(c):
		return builtin2.ActorNameByCode(c)

	case builtin3.IsBuiltinActor(c):
		return builtin3.ActorNameByCode(c)

	case builtin4.IsBuiltinActor(c):
		return builtin4.ActorNameByCode(c)

	case builtin5.IsBuiltinActor(c):
		return builtin5.ActorNameByCode(c)

	case builtin6.IsBuiltinActor(c):
		return builtin6.ActorNameByCode(c)

	case builtin7.IsBuiltinActor(c):
		return builtin7.ActorNameByCode(c)

	default:
		return "<unknown>"
	}
}

func makeAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}

func ActorFamily(name string) string {
	if name == "<unknown>" {
		return "<unknown>"
	}

	if !strings.HasPrefix(name, "fil/") {
		return "<unknown>"
	}
	idx := strings.LastIndex(name, "/")
	if idx == -1 {
		return "<unknown>"
	}

	return name[idx+1:]
}

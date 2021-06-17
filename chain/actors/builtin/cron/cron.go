// Code generated by: `make actors-gen`. DO NOT EDIT.
package cron

import (
	"github.com/ipfs/go-cid"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
)

var (
	Address = builtin5.CronActorAddr
	Methods = builtin5.MethodsCron
)

func AllCodes() []cid.Cid {
	return []cid.Cid{
		builtin0.CronActorCodeID,
		builtin2.CronActorCodeID,
		builtin3.CronActorCodeID,
		builtin4.CronActorCodeID,
		builtin5.CronActorCodeID,
	}
}

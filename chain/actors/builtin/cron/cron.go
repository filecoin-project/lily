// Code generated by: `make actors-gen`. DO NOT EDIT.
package cron

import (
	"github.com/ipfs/go-cid"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
)

var (
	Address = builtin7.CronActorAddr
	Methods = builtin7.MethodsCron
)

func AllCodes() []cid.Cid {
	return []cid.Cid{
		builtin0.CronActorCodeID,
		builtin2.CronActorCodeID,
		builtin3.CronActorCodeID,
		builtin4.CronActorCodeID,
		builtin5.CronActorCodeID,
		builtin6.CronActorCodeID,
		builtin7.CronActorCodeID,
	}
}
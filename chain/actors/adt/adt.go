package adt

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	"github.com/ipfs/go-cid"
	sha256simd "github.com/minio/sha256-simd"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

type Map interface {
	Root() (cid.Cid, error)

	Put(k abi.Keyer, v cbor.Marshaler) error
	Get(k abi.Keyer, v cbor.Unmarshaler) (bool, error)
	Delete(k abi.Keyer) error

	ForEach(v cbor.Unmarshaler, fn func(key string) error) error
}

type Array interface {
	Root() (cid.Cid, error)

	Set(idx uint64, v cbor.Marshaler) error
	Get(idx uint64, v cbor.Unmarshaler) (bool, error)
	Delete(idx uint64) error
	Length() uint64

	ForEach(v cbor.Unmarshaler, fn func(idx int64) error) error
}

type MapOpts struct {
	Bitwidth int
	HashFunc MapHashFunc
}

func (m *MapOpts) Equal(o *MapOpts) bool {
	if m.Bitwidth != o.Bitwidth {
		return false
	}

	if !bytes.Equal(m.HashFunc([]byte("string")), o.HashFunc([]byte("string"))) {
		return false
	}

	return true
}

func MapOptsForActorCode(c cid.Cid) (*MapOpts, error) {
	switch c {
	// v0
	// https://github.com/filecoin-project/specs-actors/blob/v0.9.14/actors/util/adt/map.go#L22
	case builtin0.InitActorCodeID, builtin0.StorageMarketActorCodeID, builtin0.StorageMinerActorCodeID, builtin0.MultisigActorCodeID, builtin0.StoragePowerActorCodeID, builtin0.VerifiedRegistryActorCodeID:
		return &MapOpts{
			Bitwidth: 5,
			HashFunc: Map0ShaHashFunc,
		}, nil

		// v2
		// https://github.com/filecoin-project/specs-actors/blob/v2.3.5/actors/util/adt/map.go#L22
	case builtin2.InitActorCodeID, builtin2.StorageMarketActorCodeID, builtin2.StorageMinerActorCodeID, builtin2.MultisigActorCodeID, builtin2.StoragePowerActorCodeID, builtin2.VerifiedRegistryActorCodeID:
		return &MapOpts{
			Bitwidth: 5,
			HashFunc: Map2ShaHashFunc,
		}, nil

		// v3
		// https://github.com/filecoin-project/specs-actors/blob/v3.1.1/actors/util/adt/map.go
	case builtin3.InitActorCodeID, builtin3.StorageMarketActorCodeID, builtin3.StorageMinerActorCodeID, builtin3.MultisigActorCodeID, builtin3.StoragePowerActorCodeID, builtin3.VerifiedRegistryActorCodeID:
		return &MapOpts{
			Bitwidth: builtin3.DefaultHamtBitwidth,
			HashFunc: Map2ShaHashFunc,
		}, nil

		// v4
		// https://github.com/filecoin-project/specs-actors/blob/v4.0.1/actors/util/adt/map.go#L17
	case builtin4.InitActorCodeID, builtin4.StorageMarketActorCodeID, builtin4.StorageMinerActorCodeID, builtin4.MultisigActorCodeID, builtin4.StoragePowerActorCodeID, builtin4.VerifiedRegistryActorCodeID:
		return &MapOpts{
			Bitwidth: builtin4.DefaultHamtBitwidth,
			HashFunc: Map2ShaHashFunc,
		}, nil

		// v5
		// https://github.com/filecoin-project/specs-actors/blob/v5-rc-3/actors/util/adt/map.go#L17
	case builtin5.InitActorCodeID, builtin5.StorageMarketActorCodeID, builtin5.StorageMinerActorCodeID, builtin5.MultisigActorCodeID, builtin5.StoragePowerActorCodeID, builtin5.VerifiedRegistryActorCodeID:
		return &MapOpts{
			Bitwidth: builtin5.DefaultHamtBitwidth,
			HashFunc: Map2ShaHashFunc,
		}, nil

		// v6
		// https://github.com/filecoin-project/specs-actors/blob/v6/actors/util/adt/map.go#L17
	case builtin6.InitActorCodeID, builtin6.StorageMarketActorCodeID, builtin6.StorageMinerActorCodeID, builtin6.MultisigActorCodeID, builtin6.StoragePowerActorCodeID, builtin6.VerifiedRegistryActorCodeID:
		return &MapOpts{
			Bitwidth: builtin6.DefaultHamtBitwidth,
			HashFunc: Map2ShaHashFunc,
		}, nil

		// v7
		// https://github.com/filecoin-project/specs-actors/blob/v7/actors/util/adt/map.go#L17
	case builtin7.InitActorCodeID, builtin7.StorageMarketActorCodeID, builtin7.StorageMinerActorCodeID, builtin7.MultisigActorCodeID, builtin7.StoragePowerActorCodeID, builtin7.VerifiedRegistryActorCodeID:
		return &MapOpts{
			Bitwidth: builtin7.DefaultHamtBitwidth,
			HashFunc: Map2ShaHashFunc,
		}, nil
	}

	return nil, fmt.Errorf("actor code unknown or doesn't have Map: %s", c)
}

type MapHashFunc func([]byte) []byte

var Map0ShaHashFunc MapHashFunc = func(input []byte) []byte {
	res := sha256simd.Sum256(input)
	return res[:]
}

var Map2ShaHashFunc MapHashFunc = func(input []byte) []byte {
	res := sha256.Sum256(input)
	return res[:]
}

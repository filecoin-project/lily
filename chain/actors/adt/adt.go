package adt

import (
	"bytes"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
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

type MapHashFunc func([]byte) []byte

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

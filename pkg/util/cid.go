package util

import (
	"bytes"

	"github.com/filecoin-project/go-state-types/abi"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
)

func CidOf(v typegen.CBORMarshaler) (cid.Cid, error) {
	buf := new(bytes.Buffer)
	if err := v.MarshalCBOR(buf); err != nil {
		return cid.Undef, err
	}
	c, err := abi.CidBuilder.Sum(buf.Bytes())
	if err != nil {
		return cid.Undef, err
	}
	b, err := block.NewBlockWithCid(buf.Bytes(), c)
	if err != nil {
		return cid.Undef, err
	}
	return b.Cid(), nil
}

func MustCidOf(v typegen.CBORMarshaler) cid.Cid {
	c, err := CidOf(v)
	if err != nil {
		panic(err)
	}
	return c
}

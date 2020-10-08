package testutil

import (
	"math/rand"
	"strconv"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

var cidPref = cid.Prefix{
	Version:  1,
	Codec:    cid.Raw,
	MhType:   multihash.SHA2_256,
	MhLength: -1,
}

func RandomCid() cid.Cid {
	c, err := cidPref.Sum([]byte(strconv.Itoa(rand.Int())))
	if err != nil {
		panic("randomCid: " + err.Error())
	}
	return c
}

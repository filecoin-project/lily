package v2

import (
	"github.com/filecoin-project/go-state-types/cbor"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

type LilyModel interface {
	Serialize() ([]byte, error)
	ToStorageBlock() (blocks.Block, error)
	Cid() cid.Cid
	Key() string
	cbor.Er
}

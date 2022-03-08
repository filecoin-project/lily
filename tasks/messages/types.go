package messages

import "github.com/ipfs/go-cid"

type MessageError struct {
	Cid   cid.Cid
	Error string
}

package fcjson

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ipfs/go-cid"
	"github.com/polydawn/refmt/shared"
	"github.com/polydawn/refmt/tok"

	ipld "github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/lily/tasks/messages/types"
)

// This is dagjson with special pretty-print sauce.

// Marshal is a non-link-aware marshaler
func Marshal(n ipld.Node, sink shared.TokenSink) error {
	return (&DagMarshaler{}).MarshalRecursive(n, sink)
}

type Loader func(cid.Cid, ipld.Path) ipld.Node

// DagMarshaler acts like traversal.Progress, but with emission of a token stream
// over the depth-first-search traversal.
type DagMarshaler struct {
	Loader
	Path ipld.Path
}

// MarshalRecursive is a combination traversal + codec
func (d *DagMarshaler) MarshalRecursive(n ipld.Node, sink shared.TokenSink) error {
	var tk tok.Token
	switch n.Kind() {
	case ipld.Kind_Invalid:
		return fmt.Errorf("cannot traverse a node that is absent")
	case ipld.Kind_Null:
		tk.Type = tok.TNull
		_, err := sink.Step(&tk)
		return err
	case ipld.Kind_Map:
		// Emit start of map.
		tk.Type = tok.TMapOpen
		tk.Length = int(n.Length())
		if _, err := sink.Step(&tk); err != nil {
			return err
		}
		// Emit map contents (and recurse).
		for itr := n.MapIterator(); !itr.Done(); {
			k, v, err := itr.Next()
			if err != nil {
				return err
			}
			tk.Type = tok.TString
			tk.Str, err = k.AsString()
			if err != nil {
				return err
			}
			if _, ok := k.(types.RawAddress); ok {
				a, err := address.NewFromBytes([]byte(tk.Str))
				if err != nil {
					return err
				}
				tk.Str = a.String()
			} else if _, ok := k.(types.CidString); ok {
				c := cid.Undef
				if err := c.UnmarshalBinary([]byte(tk.Str)); err != nil {
					return err
				}
				tk.Str = c.String()
			}
			if _, err := sink.Step(&tk); err != nil {
				return err
			}
			next := *d
			next.Path = next.Path.AppendSegment(ipld.PathSegmentOfString(tk.Str))
			if err := next.MarshalRecursive(v, sink); err != nil {
				return err
			}
		}
		// Emit map close.
		tk.Type = tok.TMapClose
		_, err := sink.Step(&tk)
		return err
	case ipld.Kind_List:
		// Emit start of list.
		tk.Type = tok.TArrOpen
		l := n.Length()
		tk.Length = int(l)
		if _, err := sink.Step(&tk); err != nil {
			return err
		}
		// Emit list contents (and recurse).
		for i := 0; i < int(l); i++ {
			v, err := n.LookupByIndex(int64(i))
			if err != nil {
				return err
			}
			next := *d
			next.Path.AppendSegment(ipld.PathSegmentOfInt(int64(i)))
			if err := next.MarshalRecursive(v, sink); err != nil {
				return err
			}
		}
		// Emit list close.
		tk.Type = tok.TArrClose
		_, err := sink.Step(&tk)
		return err
	case ipld.Kind_Bool:
		v, err := n.AsBool()
		if err != nil {
			return err
		}
		tk.Type = tok.TBool
		tk.Bool = v
		_, err = sink.Step(&tk)
		return err
	case ipld.Kind_Int:
		v, err := n.AsInt()
		if err != nil {
			return err
		}
		tk.Type = tok.TInt
		tk.Int = v
		_, err = sink.Step(&tk)
		return err
	case ipld.Kind_Float:
		v, err := n.AsFloat()
		if err != nil {
			return err
		}
		tk.Type = tok.TFloat64
		tk.Float64 = v
		_, err = sink.Step(&tk)
		return err
	case ipld.Kind_String:
		v, err := n.AsString()
		if err != nil {
			return err
		}
		if _, ok := n.(types.RawAddress); ok {
			a, err := address.NewFromBytes([]byte(v))
			if err != nil {
				return err
			}
			tk.Str = a.String()
		} else if _, ok := n.(types.CidString); ok {
			c := cid.Undef
			if err := c.UnmarshalBinary([]byte(v)); err != nil {
				return err
			}
			tk.Str = c.String()
		} else {
			tk.Str = v
		}
		tk.Type = tok.TString
		_, err = sink.Step(&tk)
		return err
	case ipld.Kind_Bytes:
		tk.Type = tok.TString
		v, err := n.AsBytes()
		if err != nil {
			return err
		}

		if _, ok := n.(types.Address); ok {
			a, err := address.NewFromBytes(v)
			if err != nil {
				return err
			}
			tk.Str = a.String()
		} else if _, ok := n.(types.BigInt); ok {
			i := big.NewInt(0)
			i.SetBytes(v)
			tk.Str = i.String()
		} else if _, ok := n.(types.BitField); ok {
			b, err := bitfield.NewFromBytes(v)
			if err != nil {
				if err = b.UnmarshalCBOR(bytes.NewBuffer(v)); err != nil {
					return err
				}
			}
			tk.Type = tok.TMapOpen
			tk.Length = 3
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Type = tok.TString
			tk.Str = "_type"
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Str = "bitfield"
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Str = "elemcount"
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			elemCount, err := b.Count()
			if err != nil {
				return err
			}
			tk.Str = strconv.FormatUint(elemCount, 10)
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Str = "rle"
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			buf, err := b.MarshalJSON()
			if err != nil {
				return err
			}
			tk.Str = string(buf)
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Type = tok.TMapClose
		} else {
			tk.Str = base64.StdEncoding.EncodeToString(v)
		}
		_, err = sink.Step(&tk)
		return err
	case ipld.Kind_Link:
		v, err := n.AsLink()
		if err != nil {
			return err
		}
		switch lnk := v.(type) {
		case cidlink.Link:
			if d.Loader != nil {
				node := d.Loader(lnk.Cid, d.Path)

				if node != nil {
					next := *d
					return next.MarshalRecursive(node, sink)
				}
			}
			// Precisely four tokens to emit:
			tk.Type = tok.TMapOpen
			tk.Length = 1
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Type = tok.TString
			tk.Str = "/"
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Str = lnk.Cid.String()
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			tk.Type = tok.TMapClose
			if _, err = sink.Step(&tk); err != nil {
				return err
			}
			return nil
		default:
			return fmt.Errorf("schemafree link emission only supported by this codec for CID type links")
		}
	default:
		panic("unreachable")
	}
}

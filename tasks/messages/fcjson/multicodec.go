package fcjson

import (
	"io"

	"github.com/polydawn/refmt/json"

	ipld "github.com/ipld/go-ipld-prime"
	codec "github.com/ipld/go-ipld-prime/codec"
)

var (
	// from here I think? https://github.com/ipld/go-ipld-prime/commit/a1482fe29345dd739ca3f3c3a24fe0c112d914f2#diff-d62e099e71e1d6ec60a359a5e11d26a6c07331633718c00fa37c4961a7115cb4L13-L15
	_ codec.Encoder = Encoder
)

func Encoder(n ipld.Node, w io.Writer) error {
	return Marshal(n, json.NewEncoder(w, json.EncodeOptions{
		Line:   []byte{'\n'},
		Indent: []byte{'\t'},
	}))
}

func (d *DagMarshaler) Encoder(n ipld.Node, w io.Writer) error {
	return d.MarshalRecursive(n, json.NewEncoder(w, json.EncodeOptions{
		Line:   []byte{'\n'},
		Indent: []byte{'\t'},
	}))
}

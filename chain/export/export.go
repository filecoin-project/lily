package export

import (
	"bytes"
	"context"
	"io"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-car"
	"github.com/ipld/go-car/util"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"gopkg.in/cheggaaa/pb.v1"
)

var log = logging.Logger("lily/chain/export")

type ChainExporter struct {
	store blockstore.Blockstore // blockstore chain is exported from
	head  *types.TipSet         // tipset to start export from aka CarHeader root
	out   io.Writer             // car file to export to

	min      abi.ChainEpoch // height to stop state export at
	incMg    bool           // if true export block messages
	incRt    bool           // if true export block receipts
	incSt    bool           // if true export block stateroots
	withProg bool           // if true print a progress bar.
}

func NewChainExporter(head *types.TipSet, store blockstore.Blockstore, out io.Writer) *ChainExporter {
	ce := &ChainExporter{
		store: store,
		head:  head,
		out:   out,
		min:   0,
		incMg: true,
		incRt: true,
		incSt: true,
	}
	return ce
}

func (ce *ChainExporter) Export(ctx context.Context, opts ...ExportOption) error {
	for _, opt := range opts {
		opt(ce)
	}
	log.Infow("starting chain export", "head", ce.head, "messages", ce.incMg, "receipts", ce.incRt, "stateroots", ce.incSt, "final-epoch", ce.min)
	defer log.Info("chain export complete")

	var bar *pb.ProgressBar
	if ce.withProg {
		bar = pb.New64(int64(ce.head.Height()))
		bar.ShowTimeLeft = true
		bar.ShowPercent = true
		bar.Prefix("epochs")
		bar.Units = pb.U_NO
		bar.Start()
		defer bar.Finish()
	}

	seen := cid.NewSet()
	todo := NewStack()
	h := &car.CarHeader{
		Roots:   ce.head.Cids(),
		Version: 1,
	}

	if err := car.WriteHeader(h, ce.out); err != nil {
		return err
	}
	for _, c := range ce.head.Cids() {
		todo.Push(c, BlockHeader)
	}

	for todo.Len() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		c, kind := todo.Pop()
		if !seen.Visit(c) {
			continue
		}
		if c.Prefix().Codec != cid.DagCBOR {
			continue
		}

		// we haven't visited this cid and its part of a dag, write it to file
		if err := ce.writeContent(c); err != nil {
			return err
		}

		if kind == BlockHeader {
			blk, err := ce.store.Get(c)
			if err != nil {
				return err
			}

			var b types.BlockHeader
			if err := b.UnmarshalCBOR(bytes.NewBuffer(blk.RawData())); err != nil {
				return xerrors.Errorf("unmarshaling block header (cid=%s): %w", blk, err)
			}

			if ce.withProg {
				bar.Set64(int64(ce.head.Height() - b.Height))
			}

			for _, parent := range b.Parents {
				if b.Height > 0 {
					todo.Push(parent, BlockHeader)
				} else {
					todo.Push(parent, Dag)
				}
			}

			// ensure the genesis state is always exported while skipping export for any state out
			// of range.
			if b.Height == 0 || b.Height > ce.min {
				if ce.incMg {
					if err := ce.pushNewLinks(b.Messages, todo); err != nil {
						return err
					}
				}

				if ce.incRt {
					if err := ce.pushNewLinks(b.ParentMessageReceipts, todo); err != nil {
						return err
					}
				}

				if ce.incSt {
					if err := ce.pushNewLinks(b.ParentStateRoot, todo); err != nil {
						return err
					}
				}
			}
		} else if kind == Dag {
			if err := ce.pushNewLinks(c, todo); err != nil {
				return err
			}
		} else {
			panic("received undefined cid kind")
		}
	}
	return nil
}

func (ce *ChainExporter) pushNewLinks(c cid.Cid, s *Stack) error {
	s.Push(c, Dag)
	data, err := ce.store.Get(c)
	if err != nil {
		return err
	}
	return cbg.ScanForLinks(bytes.NewReader(data.RawData()), func(visit cid.Cid) {
		s.Push(visit, Dag)
	})
}

func (ce *ChainExporter) writeContent(c cid.Cid) error {
	blk, err := ce.store.Get(c)
	if err != nil {
		return err
	}
	return util.LdWrite(ce.out, c.Bytes(), blk.RawData())
}

type ExportOption func(ce *ChainExporter)

func MinHeight(h uint64) ExportOption {
	return func(ce *ChainExporter) {
		ce.min = abi.ChainEpoch(h)
	}
}

func IncludeMessages(b bool) ExportOption {
	return func(ce *ChainExporter) {
		ce.incMg = b
	}
}

func IncludeReceipts(b bool) ExportOption {
	return func(ce *ChainExporter) {
		ce.incRt = b
	}
}

func IncludeStateRoots(b bool) ExportOption {
	return func(ce *ChainExporter) {
		ce.incSt = b
	}
}

func WithProgress(b bool) ExportOption {
	return func(ce *ChainExporter) {
		ce.withProg = b
	}
}

type cidKind int

func (c cidKind) String() string {
	if c == Undefined {
		return "Undefined"
	}
	if c == BlockHeader {
		return "BlockHeader"
	}
	if c == Dag {
		return "DAG"
	}
	return "Unknown"
}

const (
	Undefined cidKind = iota
	BlockHeader
	Dag
)

type Stack struct {
	top    *stackNode
	length int
}

type stackNode struct {
	value cid.Cid
	kind  cidKind
	prev  *stackNode
}

func NewStack() *Stack {
	return &Stack{nil, 0}
}

func (s *Stack) Len() int {
	return s.length
}

func (s *Stack) Peek() (cid.Cid, cidKind) {
	if s.length == 0 {
		return cid.Undef, Undefined
	}
	return s.top.value, s.top.kind
}

func (s *Stack) Pop() (cid.Cid, cidKind) {
	if s.length == 0 {
		return cid.Undef, Undefined
	}

	n := s.top
	s.top = n.prev
	s.length--
	return n.value, n.kind
}

func (s *Stack) Push(value cid.Cid, kind cidKind) {
	n := &stackNode{value, kind, s.top}
	s.top = n
	s.length++
}

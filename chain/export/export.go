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

	Config ExportConfig
}

func NewChainExporter(head *types.TipSet, store blockstore.Blockstore, out io.Writer, cfg ExportConfig) *ChainExporter {
	ce := &ChainExporter{
		store:  store,
		head:   head,
		out:    out,
		Config: cfg,
	}
	return ce
}

type ExportConfig struct {
	MinHeight         uint64
	IncludeMessages   bool
	IncludeReceipts   bool
	IncludeStateRoots bool
	ShowProcess       bool
}

func (ce *ChainExporter) Export(ctx context.Context) error {
	log.Infow("starting chain export", "head", ce.head, "config", ce.Config)
	defer log.Info("chain export complete")

	var bar *pb.ProgressBar
	if ce.Config.ShowProcess {
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
		if err := ce.writeContent(ctx, c); err != nil {
			return err
		}

		if kind == BlockHeader {
			blk, err := ce.store.Get(ctx, c)
			if err != nil {
				return err
			}

			var b types.BlockHeader
			if err := b.UnmarshalCBOR(bytes.NewBuffer(blk.RawData())); err != nil {
				return xerrors.Errorf("unmarshaling block header (cid=%s): %w", blk, err)
			}

			if ce.Config.ShowProcess {
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
			if b.Height == 0 || b.Height > abi.ChainEpoch(ce.Config.MinHeight) {
				if ce.Config.IncludeMessages {
					if err := ce.pushNewLinks(ctx, b.Messages, todo); err != nil {
						return err
					}
				}

				if ce.Config.IncludeReceipts {
					if err := ce.pushNewLinks(ctx, b.ParentMessageReceipts, todo); err != nil {
						return err
					}
				}

				if ce.Config.IncludeStateRoots {
					if err := ce.pushNewLinks(ctx, b.ParentStateRoot, todo); err != nil {
						return err
					}
				}
			}
		} else if kind == Dag {
			if err := ce.pushNewLinks(ctx, c, todo); err != nil {
				return err
			}
		} else {
			panic("received undefined cid kind")
		}
	}
	return nil
}

func (ce *ChainExporter) pushNewLinks(ctx context.Context, c cid.Cid, s *Stack) error {
	s.Push(c, Dag)
	data, err := ce.store.Get(ctx, c)
	if err != nil {
		return err
	}
	return cbg.ScanForLinks(bytes.NewReader(data.RawData()), func(visit cid.Cid) {
		s.Push(visit, Dag)
	})
}

func (ce *ChainExporter) writeContent(ctx context.Context, c cid.Cid) error {
	blk, err := ce.store.Get(ctx, c)
	if err != nil {
		return err
	}
	return util.LdWrite(ce.out, c.Bytes(), blk.RawData())
}

type CidKind int

func (c CidKind) String() string {
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
	Undefined CidKind = iota
	BlockHeader
	Dag
)

type Stack struct {
	top    *stackNode
	length int
}

type stackNode struct {
	value cid.Cid
	kind  CidKind
	prev  *stackNode
}

func NewStack() *Stack {
	return &Stack{nil, 0}
}

func (s *Stack) Len() int {
	return s.length
}

func (s *Stack) Peek() (cid.Cid, CidKind) {
	if s.length == 0 {
		return cid.Undef, Undefined
	}
	return s.top.value, s.top.kind
}

func (s *Stack) Pop() (cid.Cid, CidKind) {
	if s.length == 0 {
		return cid.Undef, Undefined
	}

	n := s.top
	s.top = n.prev
	s.length--
	return n.value, n.kind
}

func (s *Stack) Push(value cid.Cid, kind CidKind) {
	n := &stackNode{value, kind, s.top}
	s.top = n
	s.length++
}

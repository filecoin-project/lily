package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	v1car "github.com/ipld/go-car"
	"github.com/urfave/cli/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/filecoin-project/lily/pkg/extract/chain"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/messages"
)

func main() {
	app := &cli.App{
		Name: "transform",
		Commands: []*cli.Command{
			TransformCmd,
		},
	}
	app.Setup()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}
}

var TransformCmd = &cli.Command{
	Name: "delta",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "filename",
			Usage: "Name of the file to transform",
		},
		&cli.StringFlag{
			Name:  "db",
			Value: "host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable",
		},
	},
	Action: func(cctx *cli.Context) error {
		// Open the delta file (car file), this contains the filecoin state
		f, err := os.OpenFile(cctx.String("filename"), os.O_RDONLY, 0o644)
		defer f.Close()
		if err != nil {
			return err
		}

		// connect to the database, were gonna write stuff to this.
		db, err := gorm.Open(postgres.Open(cctx.String("db")), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix: "z_", // all tables created will be prefixed with `z_` because I am lazy and want them all in the same spot at the bottom of my table list
				// TODO figure out how to make a new schema with gorm...
			},
		})
		if err != nil {
			return err
		}

		// extract the delta state to the database and exit.
		return Run(cctx.Context, f, db)
	},
}

func Run(ctx context.Context, r io.Reader, db *gorm.DB) error {
	// create a blockstore and load the contents of the car file (reader is a pointer to the car file) into it.
	bs := blockstore.NewMemorySync()
	header, err := v1car.LoadCar(ctx, bs, r)
	if err != nil {
		return err
	}
	// expect to have a single root (cid) pointing to content
	if len(header.Roots) != 1 {
		return fmt.Errorf("invalid header expected 1 root got %d", len(header.Roots))
	}

	// we need to wrap the blockstore to meet some dumb interface.
	s := store.WrapBlockStore(ctx, bs)

	// load the root container, this contains netwwork metadata and the root (cid) of the extracted state.
	rootIPLDContainer := new(cbor.RootStateIPLD)
	if err := s.Get(ctx, header.Roots[0], rootIPLDContainer); err != nil {
		return err
	}

	// load the extracted state, this contains links to fullblocks, implicit message, network economics, and actor states
	stateExtractionIPLDContainer := new(cbor.StateExtractionIPLD)
	if err := s.Get(ctx, rootIPLDContainer.State, stateExtractionIPLDContainer); err != nil {
		return err
	}

	// ohh and it also contains the tipset the extraction was performed over.
	current := &stateExtractionIPLDContainer.Current
	parent := &stateExtractionIPLDContainer.Parent

	// for now we will only handle the fullblock dag.
	if err := HandleFullBlocks(ctx, db, s, current, parent, stateExtractionIPLDContainer.FullBlocks); err != nil {
		return err
	}

	return nil
}

func HandleFullBlocks(ctx context.Context, db *gorm.DB, s store.Store, current, parent *types.TipSet, root cid.Cid) error {
	// decode the content at the root (cid) into a concrete type we can inspect, a map of block CID to the block, its messages, their receipts and vm messages.
	fullBlocks, err := messages.DecodeFullBlockHAMT(ctx, s, root)
	if err != nil {
		return err
	}
	// migrate the database, this creates new tables or updates existing ones with new fields
	if err := db.AutoMigrate(&BlockHeaderModel{}, &ChainMessage{}, &ChainMessageReceipt{}); err != nil {
		return err
	}
	// make some blockheaders and plop em in the database
	if err := db.Create(MakeBlockHeaderModels(ctx, fullBlocks)).Error; err != nil {
		return err
	}
	// now do messages
	if err := db.Create(MakeMessages(ctx, fullBlocks)).Error; err != nil {
		return err
	}
	// finally receipts
	if err := db.Create(MakeReceipts(ctx, fullBlocks)).Error; err != nil {
		return err
	}

	// okay all done.
	return nil
}

type BlockHeaderModel struct {
	Cid           string `gorm:"primaryKey"`
	StateRoot     string
	Height        int64
	Miner         string
	ParentWeight  string `gorm:"bigint"`
	TimeStamp     uint64
	ForkSignaling uint64
	BaseFee       string `gorm:"bigint"`
	WinCount      int64
}

func MakeBlockHeaderModels(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) []*BlockHeaderModel {
	out := make([]*BlockHeaderModel, 0, len(fullBlocks))
	for _, fb := range fullBlocks {
		out = append(out, &BlockHeaderModel{
			Cid:           fb.Block.Cid().String(),
			StateRoot:     fb.Block.ParentStateRoot.String(),
			Height:        int64(fb.Block.Height),
			Miner:         fb.Block.Miner.String(),
			ParentWeight:  fb.Block.ParentWeight.String(),
			TimeStamp:     fb.Block.Timestamp,
			ForkSignaling: fb.Block.ForkSignaling,
			BaseFee:       fb.Block.ParentBaseFee.String(),
			WinCount:      fb.Block.ElectionProof.WinCount,
		})
	}
	return out
}

type ChainMessage struct {
	Cid        string `gorm:"primaryKey"`
	Version    int64
	To         string
	From       string
	Nonce      uint64
	Value      string `gorm:"bigint"`
	GasLimit   int64
	GasFeeCap  string `gorm:"bigint"`
	GasPremium string `gorm:"bigint"`
	Method     uint64
	Params     []byte
}

func MakeMessages(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) []*ChainMessage {
	// messages can be contained in more than 1 block, this is used to prevent persisting them twice
	seen := cid.NewSet()
	var out []*ChainMessage
	for _, fb := range fullBlocks {
		for _, smsg := range fb.SecpMessages {
			if !seen.Visit(smsg.Message.Cid()) {
				continue
			}
			out = append(out, &ChainMessage{
				Cid:        smsg.Message.Cid().String(),
				Version:    int64(smsg.Message.Message.Version),
				To:         smsg.Message.Message.To.String(),
				From:       smsg.Message.Message.From.String(),
				Nonce:      smsg.Message.Message.Nonce,
				Value:      smsg.Message.Message.Value.String(),
				GasLimit:   smsg.Message.Message.GasLimit,
				GasFeeCap:  smsg.Message.Message.GasFeeCap.String(),
				GasPremium: smsg.Message.Message.GasPremium.String(),
				Method:     uint64(smsg.Message.Message.Method),
				Params:     smsg.Message.Message.Params,
			})
		}
		for _, msg := range fb.BlsMessages {
			if !seen.Visit(msg.Message.Cid()) {
				continue
			}
			out = append(out, &ChainMessage{
				Cid:        msg.Message.Cid().String(),
				Version:    int64(msg.Message.Version),
				To:         msg.Message.To.String(),
				From:       msg.Message.From.String(),
				Nonce:      msg.Message.Nonce,
				Value:      msg.Message.Value.String(),
				GasLimit:   msg.Message.GasLimit,
				GasFeeCap:  msg.Message.GasFeeCap.String(),
				GasPremium: msg.Message.GasPremium.String(),
				Method:     uint64(msg.Message.Method),
				Params:     msg.Message.Params,
			})
		}
	}
	return out
}

type ChainMessageReceipt struct {
	MessageCid string `gorm:"primaryKey"`
	Index      int64  `gorm:"primaryKey"`
	ExitCode   int64
	GasUsed    int64
	Return     []byte

	BaseFeeBurn        string `gorm:"bigint"`
	OverEstimationBurn string `gorm:"bigint"`
	MinerPenalty       string `gorm:"bigint"`
	MinerTip           string `gorm:"bigint"`
	Refund             string `gorm:"bigint"`
	GasRefund          int64
	GasBurned          int64
}

func MakeReceipts(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) []*ChainMessageReceipt {
	var out []*ChainMessageReceipt
	for _, fb := range fullBlocks {
		for _, smsg := range fb.SecpMessages {
			if smsg.Receipt.GasOutputs == nil {
				continue
			}
			out = append(out, &ChainMessageReceipt{
				MessageCid:         smsg.Message.Cid().String(),
				Index:              smsg.Receipt.Index,
				ExitCode:           int64(smsg.Receipt.Receipt.ExitCode),
				GasUsed:            smsg.Receipt.Receipt.GasUsed,
				Return:             smsg.Receipt.Receipt.Return,
				BaseFeeBurn:        smsg.Receipt.GasOutputs.BaseFeeBurn.String(),
				OverEstimationBurn: smsg.Receipt.GasOutputs.OverEstimationBurn.String(),
				MinerPenalty:       smsg.Receipt.GasOutputs.MinerPenalty.String(),
				MinerTip:           smsg.Receipt.GasOutputs.MinerTip.String(),
				Refund:             smsg.Receipt.GasOutputs.Refund.String(),
				GasRefund:          smsg.Receipt.GasOutputs.GasRefund,
				GasBurned:          smsg.Receipt.GasOutputs.GasBurned,
			})
		}
		for _, msg := range fb.BlsMessages {
			if msg.Receipt.GasOutputs == nil {
				continue
			}
			out = append(out, &ChainMessageReceipt{
				MessageCid:         msg.Message.Cid().String(),
				Index:              msg.Receipt.Index,
				ExitCode:           int64(msg.Receipt.Receipt.ExitCode),
				GasUsed:            msg.Receipt.Receipt.GasUsed,
				Return:             msg.Receipt.Receipt.Return,
				BaseFeeBurn:        msg.Receipt.GasOutputs.BaseFeeBurn.String(),
				OverEstimationBurn: msg.Receipt.GasOutputs.OverEstimationBurn.String(),
				MinerPenalty:       msg.Receipt.GasOutputs.MinerPenalty.String(),
				MinerTip:           msg.Receipt.GasOutputs.MinerTip.String(),
				Refund:             msg.Receipt.GasOutputs.Refund.String(),
				GasRefund:          msg.Receipt.GasOutputs.GasRefund,
				GasBurned:          msg.Receipt.GasOutputs.GasBurned,
			})
		}
	}
	return out
}

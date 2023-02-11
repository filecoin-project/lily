package gorm

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	cid "github.com/ipfs/go-cid"
	v1car "github.com/ipld/go-car"
	"gorm.io/gorm"

	"github.com/filecoin-project/lily/pkg/extract/chain"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/messages"
	"github.com/filecoin-project/lily/pkg/transform/gorm/models"
	dbtypes "github.com/filecoin-project/lily/pkg/transform/gorm/types"
)

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
	if err := db.AutoMigrate(&models.BlockHeaderModel{}, &models.Message{}, models.MessageReceipt{}, models.VmMessage{}); err != nil {
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

	if err := db.Create(MakeVmMessages(ctx, fullBlocks)).Error; err != nil {
		return err
	}

	// okay all done.
	return nil
}

func MakeBlockHeaderModels(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) []*models.BlockHeaderModel {
	out := make([]*models.BlockHeaderModel, 0, len(fullBlocks))
	for _, fb := range fullBlocks {
		out = append(out, &models.BlockHeaderModel{
			Cid:           dbtypes.DbCID{CID: fb.Block.Cid()},
			StateRoot:     dbtypes.DbCID{CID: fb.Block.ParentStateRoot},
			Height:        int64(fb.Block.Height),
			Miner:         dbtypes.DbAddr{Addr: fb.Block.Miner},
			ParentWeight:  dbtypes.DbBigInt{BigInt: fb.Block.ParentWeight},
			TimeStamp:     fb.Block.Timestamp,
			ForkSignaling: fb.Block.ForkSignaling,
			BaseFee:       dbtypes.DbToken{Token: fb.Block.ParentBaseFee},
			WinCount:      fb.Block.ElectionProof.WinCount,
			ParentBaseFee: dbtypes.DbToken{Token: fb.Block.ParentBaseFee},
		})
	}
	return out
}

func MakeMessages(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) []*models.Message {
	// messages can be contained in more than 1 block, this is used to prevent persisting them twice
	seen := cid.NewSet()
	var out []*models.Message
	for _, fb := range fullBlocks {
		for _, smsg := range fb.SecpMessages {
			if !seen.Visit(smsg.Message.Cid()) {
				continue
			}
			out = append(out, &models.Message{
				Cid:        dbtypes.DbCID{CID: smsg.Message.Cid()},
				Version:    int64(smsg.Message.Message.Version),
				To:         dbtypes.DbAddr{Addr: smsg.Message.Message.To},
				From:       dbtypes.DbAddr{smsg.Message.Message.From},
				Nonce:      smsg.Message.Message.Nonce,
				Value:      dbtypes.DbToken{Token: smsg.Message.Message.Value},
				GasLimit:   smsg.Message.Message.GasLimit,
				GasFeeCap:  dbtypes.DbToken{Token: smsg.Message.Message.GasFeeCap},
				GasPremium: dbtypes.DbToken{Token: smsg.Message.Message.GasPremium},
				Method:     uint64(smsg.Message.Message.Method),
				Params:     smsg.Message.Message.Params,
			})
		}
		for _, msg := range fb.BlsMessages {
			if !seen.Visit(msg.Message.Cid()) {
				continue
			}
			out = append(out, &models.Message{
				Cid:        dbtypes.DbCID{CID: msg.Message.Cid()},
				Version:    int64(msg.Message.Version),
				To:         dbtypes.DbAddr{Addr: msg.Message.To},
				From:       dbtypes.DbAddr{msg.Message.From},
				Nonce:      msg.Message.Nonce,
				Value:      dbtypes.DbToken{Token: msg.Message.Value},
				GasLimit:   msg.Message.GasLimit,
				GasFeeCap:  dbtypes.DbToken{Token: msg.Message.GasFeeCap},
				GasPremium: dbtypes.DbToken{Token: msg.Message.GasPremium},
				Method:     uint64(msg.Message.Method),
				Params:     msg.Message.Params,
			})
		}
	}
	return out
}

func MakeReceipts(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) []*models.MessageReceipt {
	var out []*models.MessageReceipt
	for _, fb := range fullBlocks {
		for _, smsg := range fb.SecpMessages {
			// TODO this is buggy
			if smsg.Receipt.GasOutputs == nil {
				continue
			}
			out = append(out, &models.MessageReceipt{
				MessageCid: dbtypes.DbCID{CID: smsg.Message.Cid()},
				Receipt: models.Receipt{
					Index:    smsg.Receipt.Index,
					ExitCode: int64(smsg.Receipt.Receipt.ExitCode),
					GasUsed:  smsg.Receipt.Receipt.GasUsed,
					Return:   smsg.Receipt.Receipt.Return,
				},
				BaseFeeBurn:        dbtypes.DbToken{Token: smsg.Receipt.GasOutputs.BaseFeeBurn},
				OverEstimationBurn: dbtypes.DbToken{Token: smsg.Receipt.GasOutputs.OverEstimationBurn},
				MinerPenalty:       dbtypes.DbToken{Token: smsg.Receipt.GasOutputs.MinerPenalty},
				MinerTip:           dbtypes.DbToken{Token: smsg.Receipt.GasOutputs.MinerTip},
				Refund:             dbtypes.DbToken{Token: smsg.Receipt.GasOutputs.Refund},
				GasRefund:          smsg.Receipt.GasOutputs.GasRefund,
				GasBurned:          smsg.Receipt.GasOutputs.GasBurned,
			})
		}
		for _, msg := range fb.BlsMessages {
			if msg.Receipt.GasOutputs == nil {
				continue
			}
			out = append(out, &models.MessageReceipt{
				MessageCid: dbtypes.DbCID{CID: msg.Message.Cid()},
				Receipt: models.Receipt{
					Index:    msg.Receipt.Index,
					ExitCode: int64(msg.Receipt.Receipt.ExitCode),
					GasUsed:  msg.Receipt.Receipt.GasUsed,
					Return:   msg.Receipt.Receipt.Return,
				},
				BaseFeeBurn:        dbtypes.DbToken{Token: msg.Receipt.GasOutputs.BaseFeeBurn},
				OverEstimationBurn: dbtypes.DbToken{Token: msg.Receipt.GasOutputs.OverEstimationBurn},
				MinerPenalty:       dbtypes.DbToken{Token: msg.Receipt.GasOutputs.MinerPenalty},
				MinerTip:           dbtypes.DbToken{Token: msg.Receipt.GasOutputs.MinerTip},
				Refund:             dbtypes.DbToken{Token: msg.Receipt.GasOutputs.Refund},
				GasRefund:          msg.Receipt.GasOutputs.GasRefund,
				GasBurned:          msg.Receipt.GasOutputs.GasBurned,
			})
		}
	}
	return out
}

func MakeVmMessages(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) []*models.VmMessage {
	var out []*models.VmMessage
	for _, fb := range fullBlocks {
		for _, msg := range fb.SecpMessages {
			for _, vm := range msg.VmMessages {
				out = append(out, &models.VmMessage{
					Source: dbtypes.DbCID{CID: vm.Source},
					Cid:    dbtypes.DbCID{CID: vm.Message.Cid()},
					To:     dbtypes.DbAddr{Addr: vm.Message.To},
					From:   dbtypes.DbAddr{vm.Message.From},
					Value:  dbtypes.DbToken{Token: vm.Message.Value},
					Method: uint64(vm.Message.Method),
					Params: vm.Message.Params,
					Receipt: models.Receipt{
						Index:    vm.Index,
						ExitCode: int64(vm.Receipt.ExitCode),
						GasUsed:  vm.Receipt.GasUsed,
						Return:   vm.Receipt.Return,
					},
					Error: vm.Error,
				})
			}
		}
		for _, msg := range fb.BlsMessages {
			for _, vm := range msg.VmMessages {
				out = append(out, &models.VmMessage{
					Source: dbtypes.DbCID{CID: vm.Source},
					Cid:    dbtypes.DbCID{CID: vm.Message.Cid()},
					To:     dbtypes.DbAddr{Addr: vm.Message.To},
					From:   dbtypes.DbAddr{vm.Message.From},
					Value:  dbtypes.DbToken{Token: vm.Message.Value},
					Method: uint64(vm.Message.Method),
					Params: vm.Message.Params,
					Receipt: models.Receipt{
						Index:    vm.Index,
						ExitCode: int64(vm.Receipt.ExitCode),
						GasUsed:  vm.Receipt.GasUsed,
						Return:   vm.Receipt.Return,
					},
					Error: vm.Error,
				})
			}
		}
	}
	return out
}

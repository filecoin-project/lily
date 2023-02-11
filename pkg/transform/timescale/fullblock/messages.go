package fullblock

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lily/pkg/extract/chain"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

func ExtractMessages(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) model.Persistable {
	report := data.StartProcessingReport(tasktype.Message, mustMakeTipsetFromFullBlocks(fullBlocks))
	for _, fb := range fullBlocks {
		for _, msg := range fb.SecpMessages {
			report.AddModels(&messages.Message{
				Height:     int64(fb.Block.Height),
				Cid:        msg.Message.Cid().String(),
				From:       msg.Message.Message.From.String(),
				To:         msg.Message.Message.To.String(),
				Value:      msg.Message.Message.Value.String(),
				GasFeeCap:  msg.Message.Message.GasFeeCap.String(),
				GasPremium: msg.Message.Message.GasPremium.String(),
				GasLimit:   msg.Message.Message.GasLimit,
				SizeBytes:  msg.Message.ChainLength(),
				Nonce:      msg.Message.Message.Nonce,
				Method:     uint64(msg.Message.Message.Method),
			})
		}
		for _, msg := range fb.BlsMessages {
			report.AddModels(&messages.Message{
				Height:     int64(fb.Block.Height),
				Cid:        msg.Message.Cid().String(),
				From:       msg.Message.From.String(),
				To:         msg.Message.To.String(),
				Value:      msg.Message.Value.String(),
				GasFeeCap:  msg.Message.GasFeeCap.String(),
				GasPremium: msg.Message.GasPremium.String(),
				GasLimit:   msg.Message.GasLimit,
				SizeBytes:  msg.Message.ChainLength(),
				Nonce:      msg.Message.Nonce,
				Method:     uint64(msg.Message.Method),
			})
		}
	}
	return report.Finish()
}

func ExtractVmMessages(ctx context.Context, fullBlocks map[cid.Cid]*chain.FullBlock) model.Persistable {
	report := data.StartProcessingReport(tasktype.VMMessage, mustMakeTipsetFromFullBlocks(fullBlocks))
	for _, fb := range fullBlocks {
		for _, msg := range fb.SecpMessages {
			for _, vm := range msg.VmMessages {
				report.AddModels(&messages.VMMessage{
					Height:    int64(fb.Block.Height),
					StateRoot: fb.Block.ParentStateRoot.String(),
					Cid:       vm.Message.Cid().String(),
					Source:    vm.Source.String(),
					From:      vm.Message.From.String(),
					To:        vm.Message.To.String(),
					Value:     vm.Message.Value.String(),
					Method:    uint64(vm.Message.Method),
					ActorCode: "Deprecate",
					ExitCode:  int64(vm.Receipt.ExitCode),
					GasUsed:   vm.Receipt.GasUsed,
					Params:    "TODO make bytes", //vm.Message.Params
					Returns:   "TODO make bytes", //vm.Receipt.Return
				})
			}
		}
		for _, msg := range fb.BlsMessages {
			for _, vm := range msg.VmMessages {
				report.AddModels(&messages.VMMessage{
					Height:    int64(fb.Block.Height),
					StateRoot: fb.Block.ParentStateRoot.String(),
					Cid:       vm.Message.Cid().String(),
					Source:    vm.Source.String(),
					From:      vm.Message.From.String(),
					To:        vm.Message.To.String(),
					Value:     vm.Message.Value.String(),
					Method:    uint64(vm.Message.Method),
					ActorCode: "Deprecate",
					ExitCode:  int64(vm.Receipt.ExitCode),
					GasUsed:   vm.Receipt.GasUsed,
					Params:    "TODO make bytes", //vm.Message.Params
					Returns:   "TODO make bytes", //vm.Receipt.Return
				})
			}
		}
	}
	return report.Finish()
}

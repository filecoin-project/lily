package miner

import (
	"bytes"
	"context"
	"fmt"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	minertypes "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type PoStExtractor struct{}

func (PoStExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "PoStExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "PoStExtractor.Transform")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	// short circuit genesis state, no PoSt messages in genesis blocks.
	if !ec.HasPreviousState() {
		return nil, nil
	}
	addr := a.Address.String()
	posts := make(minermodel.MinerSectorPostList, 0)

	var partitions map[uint64]miner.Partition
	loadPartitions := func(state miner.State, epoch abi.ChainEpoch) (map[uint64]miner.Partition, error) {
		info, err := state.DeadlineInfo(epoch)
		if err != nil {
			return nil, fmt.Errorf("deadline info: %w", err)
		}
		dline, err := state.LoadDeadline(info.Index)
		if err != nil {
			return nil, fmt.Errorf("load deadline: %w", err)
		}
		pmap := make(map[uint64]miner.Partition)
		if err := dline.ForEachPartition(func(idx uint64, p miner.Partition) error {
			pmap[idx] = p
			return nil
		}); err != nil {
			return nil, fmt.Errorf("foreach partition: %w", err)
		}
		return pmap, nil
	}

	processPostMsg := func(msg types.ChainMsg, rec *types.MessageReceipt) error {
		sectors := make([]uint64, 0)
		if rec == nil || rec.ExitCode.IsError() {
			return nil
		}
		params := minertypes.SubmitWindowedPoStParams{}
		if err := params.UnmarshalCBOR(bytes.NewBuffer(msg.VMMessage().Params)); err != nil {
			return fmt.Errorf("unmarshal post params: %w", err)
		}

		var err error
		// use previous miner state and tipset state since we are using parent messages
		if partitions == nil {
			partitions, err = loadPartitions(ec.PrevState, ec.PrevTs.Height())
			if err != nil {
				return fmt.Errorf("load partitions: %w", err)
			}
		}

		for _, p := range params.Partitions {
			all, err := partitions[p.Index].AllSectors()
			if err != nil {
				return fmt.Errorf("all sectors: %w", err)
			}
			proven, err := bitfield.SubtractBitField(all, p.Skipped)
			if err != nil {
				return fmt.Errorf("subtract skipped bitfield: %w", err)
			}

			if err := proven.ForEach(func(sector uint64) error {
				sectors = append(sectors, sector)
				return nil
			}); err != nil {
				return fmt.Errorf("foreach proven: %w", err)
			}
		}

		for _, s := range sectors {
			posts = append(posts, &minermodel.MinerSectorPost{
				Height:         int64(ec.PrevTs.Height()),
				MinerID:        addr,
				SectorID:       s,
				PostMessageCID: msg.Cid().String(),
			})
		}
		return nil
	}

	msgRects, err := node.TipSetMessageReceipts(ctx, a.Current, a.Executed)
	if err != nil {
		return nil, err
	}

	for _, blkMsgs := range msgRects {
		itr, err := blkMsgs.Iterator()
		if err != nil {
			return nil, err
		}
		for itr.HasNext() {
			msg, _, rec := itr.Next()
			if msg.VMMessage().To == a.Address && msg.VMMessage().Method == 5 /* miner.SubmitWindowedPoSt */ {
				if err := processPostMsg(msg, rec); err != nil {
					return nil, fmt.Errorf("process post msg: %w", err)
				}
			}
		}
	}
	return posts, nil
}

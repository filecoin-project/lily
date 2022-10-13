package miner

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	minertypes "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	miner2 "github.com/filecoin-project/lily/tasks/actorstate/miner"
)

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&PostSectorMessage{}, ExtractPost)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&PostSectorMessage{}, supportedActors)

}

var _ v2.LilyModel = (*PostSectorMessage)(nil)

type PostSectorMessage struct {
	Height         abi.ChainEpoch
	StateRoot      cid.Cid
	Miner          address.Address
	SectorNumber   abi.SectorNumber
	PostMessageCID cid.Cid
}

func (m *PostSectorMessage) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(PostSectorMessage{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *PostSectorMessage) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *PostSectorMessage) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *PostSectorMessage) ToStorageBlock() (block.Block, error) {
	data, err := m.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (m *PostSectorMessage) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractPost(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "PostSectorMessage"), zap.Inline(a))

	ec, err := miner2.NewMinerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	// short circuit genesis state, no PoSt messages in genesis blocks.
	if !ec.HasPreviousState() {
		return nil, nil
	}
	posts := make([]v2.LilyModel, 0)

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
			posts = append(posts, &PostSectorMessage{
				Height:         ec.PrevTs.Height(),
				StateRoot:      current.ParentState(),
				Miner:          a.Address,
				SectorNumber:   abi.SectorNumber(s),
				PostMessageCID: msg.Cid(),
			})
		}
		return nil
	}

	msgRects, err := api.TipSetMessageReceipts(ctx, a.Current, a.Executed)
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

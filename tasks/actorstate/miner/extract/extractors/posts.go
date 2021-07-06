package extractors

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/miner"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/extract"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

func init() {
	extract.Register(&MinerSectorPost{}, ExtractMinerPoSts)
}

func ExtractMinerPoSts(ctx context.Context, ec *extract.MinerStateExtractionContext) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "ExtractMinerPoSts")
	defer span.End()
	if !ec.HasPreviousState() {
		return nil, nil
	}
	// short circuit genesis state, no PoSt messages in genesis blocks.
	if !ec.HasPreviousState() {
		return nil, nil
	}
	addr := ec.Address.String()
	posts := make(MinerSectorPostList, 0)
	block := ec.CurrTs.Cids()[0]
	msgs, err := ec.API.ChainGetParentMessages(ctx, block)
	if err != nil {
		return nil, xerrors.Errorf("diffing miner posts: %v", err)
	}

	var partitions map[uint64]miner.Partition
	loadPartitions := func(state miner.State, epoch abi.ChainEpoch) (map[uint64]miner.Partition, error) {
		info, err := state.DeadlineInfo(epoch)
		if err != nil {
			return nil, err
		}
		dline, err := state.LoadDeadline(info.Index)
		if err != nil {
			return nil, err
		}
		pmap := make(map[uint64]miner.Partition)
		if err := dline.ForEachPartition(func(idx uint64, p miner.Partition) error {
			pmap[idx] = p
			return nil
		}); err != nil {
			return nil, err
		}
		return pmap, nil
	}

	processPostMsg := func(msg *types.Message) error {
		sectors := make([]uint64, 0)
		rcpt, err := ec.API.StateGetReceipt(ctx, msg.Cid(), ec.CurrTs.Key())
		if err != nil {
			return err
		}
		if rcpt == nil || rcpt.ExitCode.IsError() {
			return nil
		}
		params := miner.SubmitWindowedPoStParams{}
		if err := params.UnmarshalCBOR(bytes.NewBuffer(msg.Params)); err != nil {
			return err
		}

		// use previous miner state and tipset state since we are using parent messages
		if partitions == nil {
			partitions, err = loadPartitions(ec.PrevState, ec.PrevTs.Height())
			if err != nil {
				return err
			}
		}

		for _, p := range params.Partitions {
			all, err := partitions[p.Index].AllSectors()
			if err != nil {
				return err
			}
			proven, err := bitfield.SubtractBitField(all, p.Skipped)
			if err != nil {
				return err
			}

			if err := proven.ForEach(func(sector uint64) error {
				sectors = append(sectors, sector)
				return nil
			}); err != nil {
				return err
			}
		}

		for _, s := range sectors {
			posts = append(posts, &MinerSectorPost{
				Height:         int64(ec.PrevTs.Height()),
				MinerID:        addr,
				SectorID:       s,
				PostMessageCID: msg.Cid().String(),
			})
		}
		return nil
	}

	for _, msg := range msgs {
		if msg.Message.To == ec.Address && msg.Message.Method == 5 /* miner.SubmitWindowedPoSt */ {
			if err := processPostMsg(msg.Message); err != nil {
				return nil, err
			}
		}
	}
	return posts, nil
}

type MinerSectorPost struct {
	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,notnull,use_zero"`

	PostMessageCID string
}

type MinerSectorPostList []*MinerSectorPost

func (msp *MinerSectorPost) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_posts"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, msp)
}

func (ml MinerSectorPostList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorPostList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()
	if len(ml) == 0 {
		return nil
	}

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_posts"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, ml)
}

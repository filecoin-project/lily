package processor

import (
	"context"
	"github.com/filecoin-project/visor/services/indexer"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	types "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/visor/storage"

	"github.com/gocraft/work"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/trace"
)

func NewProcessor(db *storage.Database, n api.FullNode) *Processor {
	p := NewPublisher(db)
	s := NewScheduler(n, p)
	return &Processor{
		storage:   db,
		node:      n,
		scheduler: s,
		log:       logging.Logger("processor"),
	}
}

type Processor struct {
	storage *storage.Database
	node    api.FullNode

	scheduler *Scheduler

	log    *logging.ZapEventLogger
	tracer trace.Tracer

	// we will want to spcial case the processing of the genesis state.
	genesis *types.TipSet

	batchSize int

	pool        *work.WorkerPool
	tipsetQueue *work.Enqueuer
}

func (p *Processor) InitHandler(ctx context.Context, batchSize int) error {
	if err := logging.SetLogLevel("*", "debug"); err != nil {
		return err
	}

	gen, err := p.node.ChainGetGenesis(ctx)
	if err != nil {
		return err
	}

	p.genesis = gen
	p.batchSize = batchSize

	p.log.Infow("initialized processor", "genesis", gen.String())
	return nil
}

func (p *Processor) Start(ctx context.Context) {
	p.log.Info("starting processor")
	go func() {
		for {
			select {
			case <-ctx.Done():
				p.log.Info("stopping processor")
				return
			default:
				blksToProcess, err := p.collectBlocksToProcess(ctx, p.batchSize)
				if err != nil {
					panic(err)
				}

				if len(blksToProcess) == 0 {
					time.Sleep(time.Second * 30)
					continue
				}

				actorChanges, err := p.collectActorChanges(ctx, blksToProcess)
				if err != nil {
					panic(err)
				}

				if err := p.dispatchTasks(ctx, actorChanges); err != nil {
					panic(err)
				}
			}
		}
	}()
}

func (p *Processor) dispatchTasks(ctx context.Context, changes map[cid.Cid]map[types.TipSetKey][]indexer.ActorInfo) error {
	for _, mactors := range changes[builtin.StorageMinerActorCodeID] {
		p.log.Infow("Dispatching Miner Tasks", "count", len(mactors))
		for _, mactor := range mactors {
			if _, err := p.scheduler.EnqueueMinerActorJob(mactor); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Processor) collectActorChanges(ctx context.Context, blks []*types.BlockHeader) (map[cid.Cid]map[types.TipSetKey][]indexer.ActorInfo, error) {
	out := make(map[cid.Cid]map[types.TipSetKey][]indexer.ActorInfo)
	for _, blk := range blks {
		pts, err := p.node.ChainGetTipSet(ctx, types.NewTipSetKey(blk.Parents...))
		if err != nil {
			return nil, err
		}

		changes, err := p.node.StateChangedActors(ctx, pts.ParentState(), blk.ParentStateRoot)
		if err != nil {
			return nil, err
		}

		for str, act := range changes {
			addr, err := address.NewFromString(str)
			if err != nil {
				return nil, err
			}

			_, err = p.node.StateGetActor(ctx, addr, pts.Key())
			if err == types.ErrActorNotFound {
				// TODO consider tracking deleted actors
				continue
			}
			_, err = p.node.StateGetActor(ctx, addr, pts.Parents())
			if err == types.ErrActorNotFound {
				// TODO consider tracking deleted actors
				continue
			}
			// TODO track null rounds

			_, ok := out[act.Code]
			if !ok {
				out[act.Code] = map[types.TipSetKey][]indexer.ActorInfo{}
			}
			out[act.Code][pts.Key()] = append(out[act.Code][pts.Key()], indexer.ActorInfo{
				Actor:        act,
				Address:      addr,
				TipSet:       pts.Key(),
				ParentTipset: pts.Parents(),
			})
		}
	}
	return out, nil
}

func (p *Processor) collectBlocksToProcess(ctx context.Context, batch int) ([]*types.BlockHeader, error) {
	// TODO the collect and mark as processing operations need to be atomic.
	blks, err := p.storage.CollectBlocksForProcessing(ctx, batch)
	if err != nil {
		return nil, err
	}
	if err := p.storage.MarkBlocksAsProcessing(ctx, blks); err != nil {
		return nil, err
	}

	out := make([]*types.BlockHeader, len(blks))
	for idx, blk := range blks {
		blkCid, err := cid.Decode(blk.Cid)
		if err != nil {
			return nil, err
		}
		header, err := p.node.ChainGetBlock(ctx, blkCid)
		if err != nil {
			return nil, err
		}
		out[idx] = header
	}
	return out, nil
}

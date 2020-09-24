package processor

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gocraft/work"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	types "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/parmap"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/storage"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/services/indexer"
)

func NewProcessor(db *storage.Database, n lens.API) *Processor {
	// TODO I don't like how these are buried in here.
	pubCh := make(chan model.Persistable)
	p := NewPublisher(db, pubCh)
	s := NewScheduler(n, pubCh)
	return &Processor{
		storage:   db,
		node:      n,
		scheduler: s,
		publisher: p,
		log:       logging.Logger("processor"),
	}
}

type Processor struct {
	storage *storage.Database
	node    lens.API

	scheduler *Scheduler
	publisher *Publisher

	log    *logging.ZapEventLogger
	tracer trace.Tracer

	// we will want to spcial case the processing of the genesis state.
	genesis *types.TipSet

	batchSize int

	pool        *work.WorkerPool
	tipsetQueue *work.Enqueuer
}

func (p *Processor) InitHandler(ctx context.Context, batchSize int) error {
	p.publisher.Start(ctx)
	p.scheduler.Start()

	gen, err := p.node.ChainGetGenesis(ctx)
	if err != nil {
		return xerrors.Errorf("get genesis: %w", err)
	}

	p.genesis = gen
	p.batchSize = batchSize

	if _, err := p.scheduler.queueGenesisTask(gen.Key(), gen.ParentState()); err != nil {
		return xerrors.Errorf("queue genesis task: %w", err)
	}

	p.log.Infow("initialized processor", "genesis", gen.String())
	return nil
}

func (p *Processor) Start(ctx context.Context) error {
	p.log.Info("starting processor")
	// Ensure the scheduler stops the workers and associated processes before exiting.
	defer p.scheduler.Stop()
	for {
		select {
		case <-ctx.Done():
			p.log.Info("stopping processor")
			return nil
		default:
			p.recordDBStats(ctx)
			err := p.process(ctx)
			if err != nil {
				return xerrors.Errorf("process: %w", err)
			}
		}
	}
}

func (p *Processor) process(ctx context.Context) error {
	ctx, span := global.Tracer("").Start(ctx, "Processor.process")
	defer span.End()

	blksToProcess, err := p.collectBlocksToProcess(ctx, p.batchSize)
	if err != nil {
		return xerrors.Errorf("collect blocks: %w", err)
	}

	if len(blksToProcess) == 0 {
		p.log.Info("no blocks to process, waiting 30 seconds")
		time.Sleep(time.Second * 30)
		return nil
	}
	p.log.Infow("collected blocks for processing", "count", len(blksToProcess))

	actorChanges, err := p.collectActorChanges(ctx, blksToProcess)
	if err != nil {
		return xerrors.Errorf("collect actor changes: %w", err)
	}

	p.log.Infow("collected actor changes")

	if err := p.scheduler.Dispatch(actorChanges); err != nil {
		return xerrors.Errorf("dispatch: %w", err)
	}

	return nil
}

func (p *Processor) collectActorChanges(ctx context.Context, blks []*types.BlockHeader) (map[types.TipSetKey][]indexer.ActorInfo, error) {

	out := make(map[types.TipSetKey][]indexer.ActorInfo)
	var outMu sync.Mutex
	parmap.Par(25, blks, func(blk *types.BlockHeader) {
		ctx, span := global.Tracer("").Start(ctx, "Processor.collectActorChanges")
		defer span.End()

		pts, err := p.node.ChainGetTipSet(ctx, types.NewTipSetKey(blk.Parents...))
		if err != nil {
			p.log.Error(err)
			return
		}

		changes, err := p.node.StateChangedActors(ctx, pts.ParentState(), blk.ParentStateRoot)
		if err != nil {
			p.log.Error(err)
			return
		}

		for str, act := range changes {
			addr, err := address.NewFromString(str)
			if err != nil {
				p.log.Error(err)
				continue
			}

			_, err = p.node.StateGetActor(ctx, addr, pts.Key())
			if err != nil {
				if strings.Contains(err.Error(), "actor not found") {
					// TODO consider tracking deleted actors
					continue
				}
				p.log.Error(err)
				return
			}

			_, err = p.node.StateGetActor(ctx, addr, pts.Parents())
			if err != nil {
				if strings.Contains(err.Error(), "actor not found") {
					// TODO consider tracking deleted actors
					continue
				}
				p.log.Error(err)
				return
			}

			// TODO track null rounds
			outMu.Lock()
			out[pts.Key()] = append(out[pts.Key()], indexer.ActorInfo{
				Actor:           act,
				Address:         addr,
				TipSet:          pts.Key(),
				ParentTipSet:    pts.Parents(),
				ParentStateRoot: pts.ParentState(),
			})
			outMu.Unlock()
		}

	})
	return out, nil
}

func (p *Processor) collectBlocksToProcess(ctx context.Context, batch int) ([]*types.BlockHeader, error) {
	ctx, span := global.Tracer("").Start(ctx, "Processor.collectBlocksToProcess")
	defer span.End()

	blks, err := p.storage.CollectAndMarkBlocksAsProcessing(ctx, batch)
	if err != nil {
		return nil, err
	}

	out := make([]*types.BlockHeader, len(blks))
	for idx, blk := range blks {
		blkCid, err := cid.Decode(blk.Cid)
		if err != nil {
			return nil, xerrors.Errorf("decode cid: %w", err)
		}

		header, err := p.node.ChainGetBlock(ctx, blkCid)
		if err != nil {
			return nil, xerrors.Errorf("get block: %w", err)
		}
		out[idx] = header
	}
	return out, nil
}

func (p *Processor) recordDBStats(ctx context.Context) {
	pstats := p.storage.DB.PoolStats()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.ConnState, "total"))
	stats.Record(ctx, metrics.DBConns.M(int64(pstats.TotalConns)))
	ctx, _ = tag.New(ctx, tag.Update(metrics.ConnState, "idle"))
	stats.Record(ctx, metrics.DBConns.M(int64(pstats.IdleConns)))
	ctx, _ = tag.New(ctx, tag.Update(metrics.ConnState, "stale"))
	stats.Record(ctx, metrics.DBConns.M(int64(pstats.StaleConns)))
}

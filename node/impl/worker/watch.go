package worker

import (
	"context"
	"sync"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/chain"
)

var log = logging.Logger("watch-worker")

func NewWatchWorkerManager(mctx helpers.MetricsCtx, lc fx.Lifecycle) *WatchWorkerManager {
	return &WatchWorkerManager{
		WorkerID: 0,
		Workers:  make(map[int]*WatchWorker),
	}
}

type WatchWorkerManager struct {
	WorkerID  int
	Workers   map[int]*WatchWorker
	WorkersMu sync.Mutex
}

func (wm *WatchWorkerManager) NewWatchWorker(api chainNotifyAPI, obs chain.TipSetObserver, confidence int) (ww *WatchWorker) {
	wm.WorkersMu.Lock()
	defer func() {
		wm.Workers[wm.WorkerID] = ww
		wm.WorkerID++
		wm.WorkersMu.Unlock()
	}()

	return &WatchWorker{
		ID:         wm.WorkerID,
		api:        api,
		confidence: confidence,
		cache:      chain.NewTipSetCache(confidence),
		obs:        obs,
		log:        logging.Logger("watch-worker").With("ID", wm.WorkerID),
		done:       make(chan struct{}),
	}

}

func (wm *WatchWorkerManager) StartWatcher(id int) error {
	wm.WorkersMu.Lock()
	defer wm.WorkersMu.Unlock()
	watcher, ok := wm.Workers[id]
	if !ok {
		return xerrors.Errorf("stopping watcher. ID: %d not found")
	}
	return watcher.Start()
}

func (wm *WatchWorkerManager) StopWatcher(id int) error {
	wm.WorkersMu.Lock()
	defer wm.WorkersMu.Unlock()
	watcher, ok := wm.Workers[id]
	if !ok {
		return xerrors.Errorf("stopping watcher. ID: %d not found")
	}
	watcher.Stop()
	return nil
}

type chainNotifyAPI interface {
	ChainNotify(context.Context) (<-chan []*api.HeadChange, error)
}

type WatchWorker struct {
	ID int

	api        chainNotifyAPI
	confidence int
	cache      *chain.TipSetCache
	obs        chain.TipSetObserver

	log *zap.SugaredLogger

	done chan struct{}
}

func (ww *WatchWorker) Start() error {
	ctx := context.Background()
	hc, err := ww.api.ChainNotify(ctx)
	if err != nil {
		return err
	}
	go func(headChanges <-chan []*api.HeadChange) {
		for {
			select {
			case <-ww.done:
				ww.log.Info("Stop received, stopping worker")
			case <-ctx.Done():
				ww.log.Info("Context done, stopping worker")
				return
			case headEvents, ok := <-hc:
				if !ok {
					ww.log.Warn("ChainNotify channel closed, stopping worker")
					return
				}
				if err := ww.index(ctx, headEvents); err != nil {
					ww.log.Errorw("indexing head change, stopping worker", "error", err)
					return
				}
			}
		}
	}(hc)
	return nil
}

func (ww *WatchWorker) Stop() {
	ww.done <- struct{}{}
}

func (ww *WatchWorker) index(ctx context.Context, headEvents []*api.HeadChange) error {
	for _, ch := range headEvents {
		switch ch.Type {
		case store.HCCurrent:
			ww.log.Debugw("current tipset", "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			err := ww.cache.SetCurrent(ch.Val)
			if err != nil {
				log.Errorw("tipset cache set current", "error", err.Error())
			}

			// If we have a zero confidence window then we need to notify every tipset we see
			if ww.confidence == 0 {
				if err := ww.obs.TipSet(ctx, ch.Val); err != nil {
					return xerrors.Errorf("notify tipset: %w", err)
				}
			}
		case store.HCApply:
			log.Debugw("add tipset", "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			tail, err := ww.cache.Add(ch.Val)
			if err != nil {
				log.Errorw("tipset cache add", "error", err.Error())
			}

			// Send the tipset that fell out of the confidence window to the observer
			if tail != nil {
				if err := ww.obs.TipSet(ctx, tail); err != nil {
					return xerrors.Errorf("notify tipset: %w", err)
				}
			}

		case store.HCRevert:
			log.Debugw("revert tipset", "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			err := ww.cache.Revert(ch.Val)
			if err != nil {
				log.Errorw("tipset cache revert", "error", err.Error())
			}
		}
	}

	log.Debugw("tipset cache", "height", ww.cache.Height(), "tail_height", ww.cache.TailHeight(), "length", ww.cache.Len())

	return nil
}

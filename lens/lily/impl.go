package lily

import (
	"context"
	"errors"
	"sync"

	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/bufbstore"
	"github.com/filecoin-project/lotus/node/impl/common"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
	"github.com/filecoin-project/sentinel-visor/storage"
)

type LilyNodeAPI struct {
	fx.In

	full.ChainAPI
	full.StateAPI
	common.CommonAPI
	Events *events.Events
}

func (m *LilyNodeAPI) Open(ctx context.Context) (lens.API, lens.APICloser, error) {
	return m, nil, nil
}

func (m *LilyNodeAPI) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	return util.GetExecutedMessagesForTipset(ctx, m.ChainAPI.Chain, ts, pts)
}

func (m *LilyNodeAPI) Store() adt.Store {
	bs := m.ChainAPI.Chain.Blockstore()
	cachedStore := bufbstore.NewBufferedBstore(bs)
	cs := cbor.NewCborStore(cachedStore)
	adtStore := adt.WrapStore(context.TODO(), cs)
	return adtStore
}

func (m *LilyNodeAPI) LilyWatchStart(ctx context.Context, cfg *LilyWatchConfig) error {
	log.Info("starting sentinel watch")

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	db, err := setupDatabase(ctx, cfg.Database)
	if err != nil {
		return err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the db.
	indexer, err := chain.NewTipSetIndexer(m, db, cfg.Window, cfg.Name, cfg.Tasks)
	if err != nil {
		return err
	}

	// HeadNotifier bridges between the event system and the watcher
	obs := &HeadNotifier{}

	// get the current head and set it on the tipset cache (mimic chain.watcher behaviour)
	head, err := m.ChainModuleAPI.ChainHead(ctx)
	if err != nil {
		return err
	}

	// Need to set current tipset concurrently because it will block otherwise
	go func() {
		if err := obs.SetCurrent(ctx, head); err != nil {
			log.Errorw("failed to set current head tipset", "error", err)
		}
	}()

	// TODO: rework this when we introduce job management. For the moment we start the watcher but have no way of
	// stopping it
	watcher := chain.NewWatcher(indexer, obs, cfg.Confidence)
	go func() {
		if err := watcher.Run(ctx); err != nil {
			log.Errorw("watcher stopped", "error", err)
		}
	}()

	// Hook up the notifier to the event system
	if err := m.Events.Observe(obs); err != nil {
		return err
	}

	return nil
}

func setupDatabase(ctx context.Context, cfg *LilyDatabaseConfig) (*storage.Database, error) {
	db, err := storage.NewDatabase(ctx, cfg.URL, cfg.PoolSize, cfg.Name, cfg.AllowUpsert)
	if err != nil {
		return nil, xerrors.Errorf("new database: %w", err)
	}

	if err := db.Connect(ctx); err != nil {
		if !errors.Is(err, storage.ErrSchemaTooOld) || !cfg.AllowSchemaMigration {
			return nil, xerrors.Errorf("connect database: %w", err)
		}

		log.Infof("connect database: %v", err.Error())

		// Schema is out of data and we're allowed to do schema migrations
		log.Info("Migrating schema to latest version")
		err := db.MigrateSchema(ctx)
		if err != nil {
			return nil, xerrors.Errorf("migrate schema: %w", err)
		}

		// Try to connect again
		if err := db.Connect(ctx); err != nil {
			return nil, xerrors.Errorf("connect database: %w", err)
		}
	}

	// Make sure the schema is a compatible with what this version of Visor requires
	if err := db.VerifyCurrentSchema(ctx); err != nil {
		db.Close(ctx) // nolint: errcheck
		return nil, xerrors.Errorf("verify schema: %w", err)
	}

	return db, nil
}

var _ LilyAPI = &LilyNodeAPI{}

type HeadNotifier struct {
	mu     sync.Mutex            // protects following fields
	events chan *chain.HeadEvent // created lazily, closed by first cancel call
	err    error                 // set to non-nil by the first cancel call
}

func (h *HeadNotifier) eventsCh() chan *chain.HeadEvent {
	// caller must hold mu
	if h.events == nil {
		h.events = make(chan *chain.HeadEvent)
	}
	return h.events
}

func (h *HeadNotifier) HeadEvents() <-chan *chain.HeadEvent {
	h.mu.Lock()
	ev := h.eventsCh()
	h.mu.Unlock()
	return ev
}

func (h *HeadNotifier) Err() error {
	h.mu.Lock()
	err := h.err
	h.mu.Unlock()
	return err
}

func (h *HeadNotifier) Cancel(err error) {
	h.mu.Lock()
	if h.err != nil {
		h.mu.Unlock()
		return
	}
	h.err = err
	if h.events == nil {
		h.events = make(chan *chain.HeadEvent)
	}
	close(h.events)
	h.mu.Unlock()
}

func (h *HeadNotifier) SetCurrent(ctx context.Context, ts *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventCurrent,
		TipSet: ts,
	}
	return nil
}

func (h *HeadNotifier) Apply(ctx context.Context, ts *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventApply,
		TipSet: ts,
	}
	return nil
}

func (h *HeadNotifier) Revert(ctx context.Context, ts *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventRevert,
		TipSet: ts,
	}
	return nil
}

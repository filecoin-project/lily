package lily

import (
	"context"
	"errors"

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

	// instantiate a tipset cache based on our confidence window
	tsCache := chain.NewTipSetCache(cfg.Confidence)

	// get the current head and set it on the tipset cache (mimic chain.watcher behaviour)
	head, err := m.ChainModuleAPI.ChainHead(ctx)
	if err != nil {
		return err
	}

	if err := tsCache.SetCurrent(head); err != nil {
		return err
	}

	// If we have a zero confidence window then we need to notify every tipset we see
	if cfg.Confidence == 0 {
		if err := indexer.TipSet(ctx, head); err != nil {
			return err
		}
	}

	obs := chain.NewIndexingTipSetObserver(indexer, tsCache)

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
		db.Close(ctx)
		return nil, xerrors.Errorf("verify schema: %w", err)
	}

	return db, nil
}

var _ LilyAPI = &LilyNodeAPI{}

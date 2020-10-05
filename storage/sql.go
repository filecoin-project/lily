package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/go-pg/pg/v10/types"
	"github.com/go-pg/pgext"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model/actors/common"
	init_ "github.com/filecoin-project/sentinel-visor/model/actors/init"
	"github.com/filecoin-project/sentinel-visor/model/actors/market"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/filecoin-project/sentinel-visor/model/actors/power"
	"github.com/filecoin-project/sentinel-visor/model/actors/reward"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	"github.com/filecoin-project/sentinel-visor/model/messages"
	"github.com/filecoin-project/sentinel-visor/model/visor"
)

var timeNow = time.Now

var models = []interface{}{
	(*blocks.BlockHeader)(nil),
	(*blocks.BlockParent)(nil),

	(*blocks.DrandEntrie)(nil),
	(*blocks.DrandBlockEntrie)(nil),

	(*miner.MinerPower)(nil),
	(*miner.MinerState)(nil),
	(*miner.MinerDealSector)(nil),
	(*miner.MinerSectorInfo)(nil),
	(*miner.MinerPreCommitInfo)(nil),

	(*market.MarketDealProposal)(nil),
	(*market.MarketDealState)(nil),

	(*messages.Message)(nil),
	(*messages.BlockMessage)(nil),
	(*messages.Receipt)(nil),

	(*power.ChainPower)(nil),
	(*reward.ChainReward)(nil),
	(*common.Actor)(nil),
	(*common.ActorState)(nil),

	(*init_.IdAddress)(nil),

	(*visor.ProcessingStateChange)(nil),
	(*visor.ProcessingActor)(nil),
	(*visor.ProcessingMessage)(nil),
}

var log = logging.Logger("storage")

// Advisory locks
var (
	SchemaLock AdvisoryLock = 1
)

var ErrSchemaTooOld = errors.New("database schema is too old and requires migration")
var ErrSchemaTooNew = errors.New("database schema is too new for this version of visor")

func NewDatabase(ctx context.Context, url string, poolSize int) (*Database, error) {
	opt, err := pg.ParseURL(url)
	if err != nil {
		return nil, xerrors.Errorf("parse database URL: %w", err)
	}
	opt.PoolSize = poolSize

	return &Database{opt: opt}, nil
}

type Database struct {
	DB  *pg.DB
	opt *pg.Options
}

// Connect opens a connection to the database and checks that the schema is compatible the the version required
// by this version of visor. ErrSchemaTooOld is returned if the database schema is older than the current schema,
// ErrSchemaTooNew if it is newer.
func (d *Database) Connect(ctx context.Context) error {
	db, err := connect(ctx, d.opt)
	if err != nil {
		return xerrors.Errorf("connect: %w", err)
	}

	// Check if the version of the schema is compatible
	dbVersion, latestVersion, err := getSchemaVersions(ctx, db)
	if err != nil {
		_ = db.Close()
		return xerrors.Errorf("get schema versions: %w", err)
	}

	switch {
	case latestVersion < dbVersion:
		// porridge too hot
		_ = db.Close()
		return ErrSchemaTooNew
	case latestVersion > dbVersion:
		// porridge too cold
		_ = db.Close()
		return ErrSchemaTooOld
	default:
		// just right
		d.DB = db
		return nil
	}

}

func connect(ctx context.Context, opt *pg.Options) (*pg.DB, error) {
	db := pg.Connect(opt)
	db = db.WithContext(ctx)
	db.AddQueryHook(&pgext.OpenTelemetryHook{})

	// Check if connection credentials are valid and PostgreSQL is up and running.
	if err := db.Ping(ctx); err != nil {
		return nil, xerrors.Errorf("ping database: %w", err)
	}

	// Acquire a shared lock on the schema to notify other instances that we are running
	if err := SchemaLock.LockShared(ctx, db); err != nil {
		_ = db.Close()
		return nil, xerrors.Errorf("failed to acquire schema lock, possible migration in progress: %w", err)
	}

	return db, nil
}

func (d *Database) Close(ctx context.Context) error {
	// Advisory locks are automatically closed at end of session but its still good practice to close explicitly
	if err := SchemaLock.UnlockShared(ctx, d.DB); err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("failed to release schema lock: %v", err)
	}

	err := d.DB.Close()
	d.DB = nil
	return err
}

func (d *Database) UnprocessedIndexedBlocks(ctx context.Context, maxHeight, limit int) (blocks.BlocksSynced, error) {
	var blkSynced blocks.BlocksSynced
	if err := d.DB.ModelContext(ctx, &blkSynced).
		Where("height <= ?", maxHeight).
		Where("processed_at is null").
		Order("height desc").
		Limit(limit).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
}

func (d *Database) UnprocessedIndexedTipSets(ctx context.Context, maxHeight, limit int) (visor.ProcessingStateChangeList, error) {
	var blkSynced visor.ProcessingStateChangeList
	if err := d.DB.ModelContext(ctx, &blkSynced).
		Where("height <= ?", maxHeight).
		Where("claimed_until is null").
		Order("height desc").
		Limit(limit).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
}

func (d *Database) MostRecentSyncedBlock(ctx context.Context) (*blocks.BlockSynced, error) {
	var blkSynced *blocks.BlockSynced
	if err := d.DB.ModelContext(ctx, blkSynced).
		Order("height desc").
		Limit(1).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
}

func (d *Database) MostRecentAddedTipSet(ctx context.Context) (*visor.ProcessingStateChange, error) {
	blkSynced := &visor.ProcessingStateChange{}
	if err := d.DB.ModelContext(ctx, blkSynced).
		Order("height desc").
		Limit(1).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
}

func (d *Database) CollectAndMarkBlocksAsProcessing(ctx context.Context, batch int) (blocks.BlocksSynced, error) {
	var blks blocks.BlocksSynced
	processedAt := timeNow()
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if _, err := tx.QueryContext(ctx, &blks,
			`with toProcess as (
					select cid, height, rank() over (order by height) as rnk
					from blocks_synced
					where completed_at is null and
					processed_at is null and
					height > 0
				)
				select cid
				from toProcess
				where rnk <= ?
				for update skip locked`, // ensure that only a single process can select and update blocks as processing.
			batch,
		); err != nil {
			return err
		}
		for _, blk := range blks {
			if _, err := tx.ModelContext(ctx, blk).Set("processed_at = ?", processedAt).
				WherePK().
				Update(); err != nil {
				return xerrors.Errorf("marking block as processed: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return blks, nil
}

func (d *Database) MarkBlocksAsProcessed(ctx context.Context, blks visor.ProcessingStateChangeList) error {
	return d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		completedAt := timeNow()
		for _, blk := range blks {
			if _, err := tx.ModelContext(ctx, &blk).Set("completed_at = ?", completedAt).
				WherePK().
				Update(); err != nil {
				return err
			}
		}
		return nil
	})
}

// VerifyCurrentSchema compares the schema present in the database with the models used by visor
// and returns an error if they are incompatible
func (d *Database) VerifyCurrentSchema(ctx context.Context) error {
	// If we're already connected then use that connection
	if d.DB != nil {
		return verifyCurrentSchema(ctx, d.DB)
	}

	// Temporarily connect
	db, err := connect(ctx, d.opt)
	if err != nil {
		return xerrors.Errorf("connect: %w", err)
	}
	defer db.Close()
	return verifyCurrentSchema(ctx, db)
}

func verifyCurrentSchema(ctx context.Context, db *pg.DB) error {
	valid := true
	for _, model := range models {
		q := db.Model(model)
		tm := q.TableModel()
		m := tm.Table()
		err := verifyModel(ctx, db, m)
		if err != nil {
			valid = false
			log.Errorf("verify schema: %v", err)
		}

	}
	if !valid {
		return xerrors.Errorf("database schema was not compatible with current models")
	}
	return nil
}

func verifyModel(ctx context.Context, db *pg.DB, m *orm.Table) error {
	tableName := stripQuotes(m.SQLNameForSelects)

	var exists bool
	_, err := db.QueryOne(pg.Scan(&exists), `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=?)`, tableName)
	if err != nil {
		return xerrors.Errorf("querying table: %v", err)
	}
	if !exists {
		return xerrors.Errorf("required table %s not found", m.SQLName)
	}

	for _, fld := range m.Fields {
		var datatype string
		_, err := db.QueryOne(pg.Scan(&datatype), `SELECT data_type FROM information_schema.columns WHERE table_schema='public' AND table_name=? AND column_name=?`, tableName, fld.SQLName)
		if err != nil {
			if errors.Is(err, pg.ErrNoRows) {
				return xerrors.Errorf("required column %s.%s not found", tableName, fld.SQLName)
			}
			return xerrors.Errorf("querying field: %v %T", err, err)
		}

		// Some common aliases
		if datatype == "timestamp with time zone" {
			datatype = "timestamptz"
		} else if datatype == "timestamp without time zone" {
			datatype = "timestamp"
		}

		if datatype != fld.SQLType {
			return xerrors.Errorf("column %s.%s had datatype %s, expected %s", tableName, fld.SQLName, datatype, fld.SQLType)
		}

	}

	return nil
}

func stripQuotes(s types.Safe) string {
	return strings.Trim(string(s), `"`)
}

func (d *Database) LeaseStateChanges(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingStateChangeList, error) {
	var blocks visor.ProcessingStateChangeList

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &blocks, `
WITH leased AS (
    UPDATE visor_processing_statechanges
    SET claimed_until = ?
    FROM (
	    SELECT *
	    FROM visor_processing_statechanges
	    WHERE completed_at IS null AND
	          (claimed_until IS null OR claimed_until < ?) AND
	          height >= ? AND height <= ?
	    ORDER BY height DESC
	    LIMIT ?
	    FOR UPDATE SKIP LOCKED
	) candidates
	WHERE visor_processing_statechanges.tip_set = candidates.tip_set AND visor_processing_statechanges.height = candidates.height
    RETURNING visor_processing_statechanges.tip_set, visor_processing_statechanges.height
)
SELECT tip_set,height FROM leased;
    `, claimUntil, timeNow(), minHeight, maxHeight, batchSize)

		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return blocks, nil
}

func (d *Database) MarkStateChangeComplete(ctx context.Context, tsk string, height int64, completedAt time.Time, errorsDetected string) error {
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_statechanges
    SET claimed_until = null,
        completed_at = ?,
        errors_detected = ?
    WHERE tip_set = ? AND height = ?
`, completedAt, useNullIfEmpty(errorsDetected), tsk, height)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// LeaseActors leases a set of actors to process. minHeight and maxHeight define an inclusive range of heights to process.
func (d *Database) LeaseActors(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64, codes []string) (visor.ProcessingActorList, error) {
	var actors visor.ProcessingActorList

	// Ensure we never return genesis, which is handled separately
	if minHeight < 1 {
		minHeight = 1
	}

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &actors, `
WITH leased AS (
    UPDATE visor_processing_actors a
    SET claimed_until = ?
    FROM (
	    SELECT *
	    FROM visor_processing_actors
	    WHERE completed_at IS null AND
	          (claimed_until IS null OR claimed_until < ?) AND
	          height >= ? AND height <= ? AND
	          code IN (?)
	    ORDER BY height DESC
	    LIMIT ?
	    FOR UPDATE SKIP LOCKED
	) candidates
	WHERE a.head = candidates.head AND a.code = candidates.code
    RETURNING a.head, a.code, a.nonce, a.balance, a.address, a.parent_state_root, a.tip_set, a.parent_tip_set)
SELECT head, code, nonce, balance, address, parent_state_root, tip_set, parent_tip_set from leased;
    `, claimUntil, timeNow(), minHeight, maxHeight, pg.In(codes), batchSize)

		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return actors, nil
}

func (d *Database) MarkActorComplete(ctx context.Context, head string, code string, completedAt time.Time, errorsDetected string) error {
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_actors
    SET claimed_until = null,
        completed_at = ?,
        errors_detected = ?
    WHERE head = ? AND code = ?
`, completedAt, useNullIfEmpty(errorsDetected), head, code)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// LeaseActors leases a set of tipsets containing messages to process. minHeight and maxHeight define an inclusive range of heights to process.
func (d *Database) LeaseTipSetMessages(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingMessageList, error) {
	var messages visor.ProcessingMessageList

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &messages, `
WITH leased AS (
    UPDATE visor_processing_messages
    SET claimed_until = ?
    FROM (
	    SELECT *
	    FROM visor_processing_messages
	    WHERE completed_at IS null AND
	          (claimed_until IS null OR claimed_until < ?) AND
	          height >= ? AND height <= ?
	    ORDER BY height DESC
	    LIMIT ?
	    FOR UPDATE SKIP LOCKED
	) candidates
	WHERE visor_processing_messages.tip_set = candidates.tip_set AND visor_processing_messages.height = candidates.height
    RETURNING visor_processing_messages.tip_set, visor_processing_messages.height
)
SELECT tip_set,height FROM leased;
    `, claimUntil, timeNow(), minHeight, maxHeight, batchSize)

		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return messages, nil
}

func (d *Database) MarkTipSetMessagesComplete(ctx context.Context, tipset string, height int64, completedAt time.Time, errorsDetected string) error {
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_messages
    SET claimed_until = null,
        completed_at = ?,
        errors_detected = ?
    WHERE tip_set = ? AND height = ?
`, completedAt, useNullIfEmpty(errorsDetected), tipset, height)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func useNullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

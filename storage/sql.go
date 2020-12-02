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
	"github.com/raulk/clock"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/actors/common"
	init_ "github.com/filecoin-project/sentinel-visor/model/actors/init"
	"github.com/filecoin-project/sentinel-visor/model/actors/market"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/filecoin-project/sentinel-visor/model/actors/multisig"
	"github.com/filecoin-project/sentinel-visor/model/actors/power"
	"github.com/filecoin-project/sentinel-visor/model/actors/reward"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	"github.com/filecoin-project/sentinel-visor/model/chain"
	"github.com/filecoin-project/sentinel-visor/model/derived"
	"github.com/filecoin-project/sentinel-visor/model/messages"
	"github.com/filecoin-project/sentinel-visor/model/visor"
)

var models = []interface{}{
	(*blocks.BlockHeader)(nil),
	(*blocks.BlockParent)(nil),
	(*blocks.DrandBlockEntrie)(nil),

	(*miner.MinerSectorDeal)(nil),
	(*miner.MinerSectorInfo)(nil),
	(*miner.MinerSectorPost)(nil),
	(*miner.MinerPreCommitInfo)(nil),
	(*miner.MinerSectorEvent)(nil),
	(*miner.MinerCurrentDeadlineInfo)(nil),
	(*miner.MinerFeeDebt)(nil),
	(*miner.MinerLockedFund)(nil),
	(*miner.MinerInfo)(nil),

	(*market.MarketDealProposal)(nil),
	(*market.MarketDealState)(nil),

	(*messages.Message)(nil),
	(*messages.BlockMessage)(nil),
	(*messages.Receipt)(nil),
	(*messages.MessageGasEconomy)(nil),
	(*messages.ParsedMessage)(nil),

	(*multisig.MultisigTransaction)(nil),

	(*power.ChainPower)(nil),
	(*reward.ChainReward)(nil),
	(*common.Actor)(nil),
	(*common.ActorState)(nil),

	(*init_.IdAddress)(nil),

	(*visor.ProcessingTipSet)(nil),
	(*visor.ProcessingActor)(nil),
	(*visor.ProcessingMessage)(nil),

	(*visor.ProcessingStat)(nil),

	(*derived.GasOutputs)(nil),
	(*chain.ChainEconomics)(nil),
}

var log = logging.Logger("storage")

// Advisory locks
var (
	SchemaLock AdvisoryLock = 1
)

var (
	ErrSchemaTooOld = errors.New("database schema is too old and requires migration")
	ErrSchemaTooNew = errors.New("database schema is too new for this version of visor")
)

func NewDatabase(ctx context.Context, url string, poolSize int, name string) (*Database, error) {
	opt, err := pg.ParseURL(url)
	if err != nil {
		return nil, xerrors.Errorf("parse database URL: %w", err)
	}
	opt.PoolSize = poolSize
	if opt.ApplicationName == "" {
		opt.ApplicationName = name
	}

	return &Database{
		opt:   opt,
		Clock: clock.New(),
	}, nil
}

type Database struct {
	DB    *pg.DB
	opt   *pg.Options
	Clock clock.Clock
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

func (d *Database) UnprocessedIndexedTipSets(ctx context.Context, maxHeight, limit int) (visor.ProcessingTipSetList, error) {
	var blkSynced visor.ProcessingTipSetList
	if err := d.DB.ModelContext(ctx, &blkSynced).
		Where("height <= ?", maxHeight).
		Where("statechange_claimed_until is null").
		Order("height desc").
		Limit(limit).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
}

func (d *Database) MostRecentAddedTipSet(ctx context.Context) (*visor.ProcessingTipSet, error) {
	blkSynced := &visor.ProcessingTipSet{}
	if err := d.DB.ModelContext(ctx, blkSynced).
		Order("height desc").
		Limit(1).
		Select(); err != nil {
		return nil, err
	}
	return blkSynced, nil
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
	_, err := db.QueryOneContext(ctx, pg.Scan(&exists), `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=?)`, tableName)
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
		if datatype == "USER-DEFINED" {
			_, err := db.QueryOne(pg.Scan(&datatype), `SELECT udt_name FROM information_schema.columns WHERE table_schema='public' AND table_name=? AND column_name=?`, tableName, fld.SQLName)
			if err != nil {
				if errors.Is(err, pg.ErrNoRows) {
					return xerrors.Errorf("required column %s.%s not found", tableName, fld.SQLName)
				}
				return xerrors.Errorf("querying field: %v %T", err, err)
			}
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

func (d *Database) LeaseStateChanges(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingTipSetList, error) {
	stop := metrics.Timer(ctx, metrics.BatchSelectionDuration)
	defer stop()

	var blocks visor.ProcessingTipSetList

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &blocks, `
WITH leased AS (
    UPDATE visor_processing_tipsets
    SET statechange_claimed_until = ?
    FROM (
	    SELECT *
	    FROM visor_processing_tipsets
	    WHERE statechange_completed_at IS null AND
	          (statechange_claimed_until IS null OR statechange_claimed_until < ?) AND
	          height >= ? AND height <= ?
	    ORDER BY height DESC
	    LIMIT ?
	    FOR UPDATE SKIP LOCKED
	) candidates
	WHERE visor_processing_tipsets.tip_set = candidates.tip_set AND visor_processing_tipsets.height = candidates.height
	AND visor_processing_tipsets.height >= ? AND visor_processing_tipsets.height <= ?
    RETURNING visor_processing_tipsets.tip_set, visor_processing_tipsets.height
)
SELECT tip_set,height FROM leased;
    `, claimUntil, d.Clock.Now(), minHeight, maxHeight, batchSize, minHeight, maxHeight)
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
	stop := metrics.Timer(ctx, metrics.CompletionDuration)
	defer stop()
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_tipsets
    SET statechange_claimed_until = null,
        statechange_completed_at = ?,
        statechange_errors_detected = ?
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

// GetActorByHead returns an actor without a lease by its CID
func (d *Database) GetActorByHead(ctx context.Context, head string) (*visor.ProcessingActor, error) {
	if len(head) == 0 {
		return nil, xerrors.Errorf("lookup actor head was empty")
	}

	d.DB.AddQueryHook(pgext.DebugHook{
		Verbose: true, // Print all queries.
	})

	a := new(visor.ProcessingActor)
	if err := d.DB.ModelContext(ctx, a).
		Where("head = ?", head).
		Limit(1).
		Select(); err != nil {
		return nil, err
	}
	return a, nil
}

// LeaseActors leases a set of actors to process. minHeight and maxHeight define an inclusive range of heights to process.
func (d *Database) LeaseActors(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64, codes []string) (visor.ProcessingActorList, error) {
	stop := metrics.Timer(ctx, metrics.BatchSelectionDuration)
	defer stop()
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
 	AND a.height >= ? AND a.height <= ?
   RETURNING a.head, a.code, a.nonce, a.balance, a.address, a.parent_state_root, a.tip_set, a.parent_tip_set, a.height)
SELECT head, code, nonce, balance, address, parent_state_root, tip_set, parent_tip_set, height from leased;
    `, claimUntil, d.Clock.Now(), minHeight, maxHeight, pg.In(codes), batchSize, minHeight, maxHeight)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return actors, nil
}

// FindActors finds a set of actors to process but does not take a lease out. minHeight and maxHeight define an inclusive range of heights to process.
func (d *Database) FindActors(ctx context.Context, batchSize int, minHeight, maxHeight int64, codes []string) (visor.ProcessingActorList, error) {
	stop := metrics.Timer(ctx, metrics.BatchSelectionDuration)
	defer stop()
	var actors visor.ProcessingActorList

	// Ensure we never return genesis, which is handled separately
	if minHeight < 1 {
		minHeight = 1
	}

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		var err error
		switch len(codes) {
		case 0:
			_, err = tx.QueryContext(ctx, &actors, `
				    SELECT head, code, nonce, balance, address, parent_state_root, tip_set, parent_tip_set, height
				    FROM visor_processing_actors
				    WHERE completed_at IS null AND height >= ? AND height <= ?
				    ORDER BY height DESC
				    LIMIT ?`, minHeight, maxHeight, batchSize)
		case 1:
			_, err = tx.QueryContext(ctx, &actors, `
				    SELECT head, code, nonce, balance, address, parent_state_root, tip_set, parent_tip_set, height
				    FROM visor_processing_actors
				    WHERE completed_at IS null AND height >= ? AND height <= ? AND code = ?
				    ORDER BY height DESC
				    LIMIT ?`, minHeight, maxHeight, codes[0], batchSize)
		default:
			_, err = tx.QueryContext(ctx, &actors, `
				    SELECT head, code, nonce, balance, address, parent_state_root, tip_set, parent_tip_set, height
				    FROM visor_processing_actors
				    WHERE completed_at IS null AND height >= ? AND height <= ? AND code IN (?)
				    ORDER BY height DESC
				    LIMIT ?`, minHeight, maxHeight, pg.In(codes), batchSize)
		}
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return actors, nil
}

func (d *Database) MarkActorComplete(ctx context.Context, height int64, head string, code string, completedAt time.Time, errorsDetected string) error {
	stop := metrics.Timer(ctx, metrics.CompletionDuration)
	defer stop()
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_actors
    SET claimed_until = null,
        completed_at = ?,
        errors_detected = ?
    WHERE height = ? AND head = ? AND code = ?
`, completedAt, useNullIfEmpty(errorsDetected), height, head, code)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// LeaseTipSetMessages leases a set of tipsets containing messages to process. minHeight and maxHeight define an inclusive range of heights to process.
func (d *Database) LeaseTipSetMessages(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingTipSetList, error) {
	stop := metrics.Timer(ctx, metrics.BatchSelectionDuration)
	defer stop()
	var messages visor.ProcessingTipSetList

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &messages, `
WITH leased AS (
    UPDATE visor_processing_tipsets
    SET message_claimed_until = ?
    FROM (
	    SELECT *
	    FROM visor_processing_tipsets
	    WHERE message_completed_at IS null AND
	          (message_claimed_until IS null OR message_claimed_until < ?) AND
	          height >= ? AND height <= ?
	    ORDER BY height DESC
	    LIMIT ?
	    FOR UPDATE SKIP LOCKED
	) candidates
	WHERE visor_processing_tipsets.tip_set = candidates.tip_set AND visor_processing_tipsets.height = candidates.height
	AND visor_processing_tipsets.height >= ? AND visor_processing_tipsets.height <= ?
    RETURNING visor_processing_tipsets.tip_set, visor_processing_tipsets.height
)
SELECT tip_set,height FROM leased;
    `, claimUntil, d.Clock.Now(), minHeight, maxHeight, batchSize, minHeight, maxHeight)
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
	stop := metrics.Timer(ctx, metrics.CompletionDuration)
	defer stop()
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_tipsets
    SET message_claimed_until = null,
        message_completed_at = ?,
        message_errors_detected = ?
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

// LeaseGasOutputsMessages leases a set of messages that have receipts for gas output processing. minHeight and maxHeight define an inclusive range of heights to process.
func (d *Database) LeaseGasOutputsMessages(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) ([]*derived.ProcessingGasOutputs, error) {
	stop := metrics.Timer(ctx, metrics.BatchSelectionDuration)
	defer stop()
	var list []*derived.ProcessingGasOutputs

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &list, `
WITH leased AS (
    UPDATE visor_processing_messages pm
    SET gas_outputs_claimed_until = ?
    FROM (
		SELECT pm.cid, m.from, m.to, m.size_bytes, m.nonce, m.value,
			   m.gas_fee_cap, m.gas_premium, m.gas_limit, m.method,
			   r.state_root, r.exit_code,r.gas_used, bh.parent_base_fee
		FROM visor_processing_messages pm
		JOIN receipts r ON pm.cid = r.message -- don't join receipts on height since it's the height of the receipt
		JOIN messages m ON pm.cid = m.cid AND pm.height = m.height
		JOIN block_messages bm on pm.cid = bm.message AND pm.height = bm.height
		JOIN block_headers bh on bm.block = bh.cid AND bm.height = bh.height
		WHERE pm.gas_outputs_completed_at IS null AND
		      (pm.gas_outputs_claimed_until IS null OR pm.gas_outputs_claimed_until < ?) AND
		      pm.height >= ? AND pm.height <= ?
		ORDER BY pm.height DESC
		LIMIT ?
		FOR UPDATE SKIP LOCKED
	) candidates
	WHERE pm.cid = candidates.cid
	AND pm.height >= ? AND pm.height <= ?
    RETURNING pm.height, candidates.*
)
SELECT * FROM leased;
`, claimUntil, d.Clock.Now(), minHeight, maxHeight, batchSize, minHeight, maxHeight)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// FindGasOutputsMessages finds a set of messages that have receipts for gas output processing but does not take a lease out. minHeight and maxHeight define an inclusive range of heights to process.
func (d *Database) FindGasOutputsMessages(ctx context.Context, batchSize int, minHeight, maxHeight int64) ([]*derived.ProcessingGasOutputs, error) {
	stop := metrics.Timer(ctx, metrics.BatchSelectionDuration)
	defer stop()

	var list []*derived.ProcessingGasOutputs

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &list, `
		SELECT pm.height, pm.cid, m.from, m.to, m.size_bytes, m.nonce, m.value,
			   m.gas_fee_cap, m.gas_premium, m.gas_limit, m.method,
			   r.state_root, r.exit_code,r.gas_used, bh.parent_base_fee
		FROM visor_processing_messages pm
		JOIN receipts r ON pm.cid = r.message -- don't join receipts on height since it's the height of the receipt
		JOIN messages m ON pm.cid = m.cid AND pm.height = m.height
		JOIN block_messages bm on pm.cid = bm.message AND pm.height = bm.height
		JOIN block_headers bh on bm.block = bh.cid AND bm.height = bh.height
		WHERE pm.gas_outputs_completed_at IS null AND
		      pm.height >= ? AND pm.height <= ?
		ORDER BY pm.height DESC
		LIMIT ?
`, minHeight, maxHeight, batchSize)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

func (d *Database) MarkGasOutputsMessagesComplete(ctx context.Context, height int64, cid string, completedAt time.Time, errorsDetected string) error {
	stop := metrics.Timer(ctx, metrics.CompletionDuration)
	defer stop()
	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_messages
    SET gas_outputs_claimed_until = null,
        gas_outputs_completed_at = ?,
        gas_outputs_errors_detected = ?
    WHERE height = ? AND cid = ?
`, completedAt, useNullIfEmpty(errorsDetected), height, cid)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// LeaseTipSetEconomics leases a set of tipsets containing chain economics to process. minHeight and maxHeight define an inclusive range of heights to process.
// TODO: refactor all the tipset leasing methods into a more general function
func (d *Database) LeaseTipSetEconomics(ctx context.Context, claimUntil time.Time, batchSize int, minHeight, maxHeight int64) (visor.ProcessingTipSetList, error) {
	stop := metrics.Timer(ctx, metrics.BatchSelectionDuration)
	defer stop()
	var tipsets visor.ProcessingTipSetList

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.QueryContext(ctx, &tipsets, `
WITH leased AS (
    UPDATE visor_processing_tipsets
    SET economics_claimed_until = ?
    FROM (
	    SELECT *
	    FROM visor_processing_tipsets
	    WHERE economics_completed_at IS null AND
	          (economics_claimed_until IS null OR economics_claimed_until < ?) AND
	          height >= ? AND height <= ?
	    ORDER BY height DESC
	    LIMIT ?
	    FOR UPDATE SKIP LOCKED
	) candidates
	WHERE visor_processing_tipsets.tip_set = candidates.tip_set AND visor_processing_tipsets.height = candidates.height
	AND visor_processing_tipsets.height >= ? AND visor_processing_tipsets.height <= ?
    RETURNING visor_processing_tipsets.tip_set, visor_processing_tipsets.height
)
SELECT tip_set,height FROM leased;
    `, claimUntil, d.Clock.Now(), minHeight, maxHeight, batchSize, minHeight, maxHeight)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return tipsets, nil
}

func (d *Database) MarkTipSetEconomicsComplete(ctx context.Context, tipset string, height int64, completedAt time.Time, errorsDetected string) error {
	stop := metrics.Timer(ctx, metrics.CompletionDuration)
	defer stop()

	if err := d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		_, err := tx.ExecContext(ctx, `
    UPDATE visor_processing_tipsets
    SET economics_claimed_until = null,
        economics_completed_at = ?,
        economics_errors_detected = ?
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

func (d *Database) Persist(ctx context.Context, p model.PersistableWithTx) error {
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return d.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return p.PersistWithTx(ctx, tx)
	})
}

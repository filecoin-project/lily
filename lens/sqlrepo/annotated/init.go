package annotated

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/dustin/go-humanize"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	logging "github.com/ipfs/go-log/v2"
)

const (
	// magnitude of overhead is unclear - just keep it on for now
	withCacheMetrics = true

	envVarConn      = "LOTUS_CHAINSTORE_PG_CONNSTRING"
	envVarCacheGiB  = "LOTUS_CHAINSTORE_CACHE_SIZE"
	envVarLogAccess = "LOTUS_CHAINSTORE_LOG_ACCESS"

	trackRecentTipsets = 950 // a bit over finality
	pgCursorStride     = 8192

	// Applies to github.com/valyala/gozstd only
	// Not using klauspost's pure-go version for compression for now
	zstdCompressLevel = 11
)

var log = logging.Logger("annotated-blockstore")

func NewPgChainStore(ctx context.Context, connectString string) (Chainstore, error) {

	cacheSize := int64(16 << 30) // 16GiB of cache by default, overridable through envVarCacheGiB

	if connectString == "" {
		var envSet bool
		connectString, envSet = os.LookupEnv(envVarConn)
		if !envSet || connectString == "" {
			return nil, fmt.Errorf(
				"\n\nyou must set the '%s' environment variable to a valid PostgreSQL connection string, e.g. 'postgres:///{{dbname}}?user={{user}}&password={{pass}}&host=/var/run/postgresql'",
				envVarConn,
			)
		}
	}

	dbPool, err := pgxpool.Connect(ctx, connectString)
	if err != nil {
		return nil, xerrors.Errorf("failed to connect to '%s': %w", connectString, err)
	}

	dbSettingsExpected := "Encoding:SQL_ASCII Collate:C Ctype:C"
	var dbSettings string
	if err := dbPool.QueryRow(
		ctx,
		`
		SELECT 'Encoding:' || pg_encoding_to_char(encoding) || ' Collate:' || datcollate || ' Ctype:' || datctype
			FROM pg_database
		WHERE datname = current_database()
		`,
	).Scan(&dbSettings); err != nil {
		return nil, err
	}
	if dbSettings != dbSettingsExpected {
		return nil, fmt.Errorf(
			"unexpected database settings: you must create your database with something like `%s` for reliable and performant binary storage\n Current settings: %s\nExpected settings: %s",
			`CREATE DATABASE {{name}} ENCODING='SQL_ASCII' LC_COLLATE='C' LC_CTYPE='C' TEMPLATE='template0'`,
			dbSettings, dbSettingsExpected,
		)
	}

	if customCacheSizeStr := os.Getenv(envVarCacheGiB); customCacheSizeStr != "" {
		gibSize, err := strconv.ParseUint(customCacheSizeStr, 10, 8)
		if err != nil {
			return nil, xerrors.Errorf("failed to parse cache size '%s': %w", customCacheSizeStr, err)
		}
		cacheSize = int64(gibSize) << 30
	}

	// cache stores blockUnit's keyed by b.Cid().Bytes()
	cache, _ := ristretto.NewCache(&ristretto.Config{
		Metrics:     withCacheMetrics,
		NumCounters: 128 << 20, // 128 million counters ~~ 10 million individual blocks
		MaxCost:     cacheSize,
		BufferItems: 64,
		Cost: func(interface{}) int64 {
			log.Panic("cost estimator should never have been called - every cache.Set() must be properly sized")
			return -1
		},
	})

	cs := &acs{
		dbPool:               dbPool,
		cache:                cache,
		cacheSize:            cacheSize,
		accessStatsRecent:    make(map[uint64]struct{}, 16384),
		limiterSetLastAccess: make(chan struct{}, 1),
		limiterBlockParse:    make(chan struct{}, runtime.NumCPU()),
		limiterCompress:      make(chan struct{}, runtime.NumCPU()),
	}

	if logAccess := os.Getenv(envVarLogAccess); logAccess != "" && logAccess != "0" && logAccess != "false" {
		cs.accessStatsHiRes = make(map[accessUnit]uint64, 65535)
	}

	if err := cs.deploy(ctx); err != nil {
		return nil, err
	}

	// Wire-up a cache stats logger triggered on USR1
	sigChUSR1 := make(chan os.Signal, 1)
	if withCacheMetrics {

		go func() {
			// infloop until shutdown
			for {
				<-sigChUSR1

				var pctFull float64
				curSize := cache.Metrics.CostAdded() - cache.Metrics.CostEvicted()
				if curSize != 0 {
					pctFull = float64(curSize) * 100 / float64(cacheSize)
				}

				log.Infof(`
--- BLOCK CACHE STATS
Current Entries: % 12s
Current Bytes: % 14s  (%0.2f%% full)
Hit Ratio: %.02f%%
Hits:% 12s Misses:% 12s
Sets:% 12s  Drops:% 12s`,
					humanize.Comma(int64(cache.Metrics.KeysAdded()-cache.Metrics.KeysEvicted())),
					humanize.Comma(int64(curSize)), pctFull,
					cache.Metrics.Ratio()*100,
					humanize.Comma(int64(cache.Metrics.Hits())), humanize.Comma(int64(cache.Metrics.Misses())),
					humanize.Comma(int64(cache.Metrics.KeysAdded())), humanize.Comma(int64(cache.Metrics.KeysEvicted())),
				)
			}
		}()
		signal.Notify(sigChUSR1, syscall.SIGUSR1)
	}

	// Wire-up a cache-purge triggered on USR2
	sigChUSR2 := make(chan os.Signal, 1)
	go func() {
		for {
			<-sigChUSR2
			cache.Clear()

			if withCacheMetrics {
				// trigger stats print if possible
				select {
				case sigChUSR1 <- syscall.SIGUSR1:
				default:
				}

				time.Sleep(10 * time.Millisecond)
				cache.Metrics.Clear()

				select {
				case sigChUSR1 <- syscall.SIGUSR1:
				default:
				}
			}
		}
	}()
	signal.Notify(sigChUSR2, syscall.SIGUSR2)

	// Background initial state prefetch
	// It panic()s on error
	go func() {

		ctx := context.Background()

		t0 := time.Now()
		var wg sync.WaitGroup
		totalCount := new(int64)
		totalBytes := new(int64)

		for _, sel := range []string{
			`
			SELECT blkid, cid, size, compressed_content
				FROM blocks_content bc
			WHERE	blkid IN ( SELECT header_blkid FROM tipsets_headers )
			`,

			`
			SELECT blkid, cid, size, compressed_content
				FROM blocks_content bc
			WHERE blkid IN ( SELECT blkid FROM blocks_recent )
			`,
		} {

			wg.Add(1)

			go func(selector string) {

				tx, err := dbPool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
				if err != nil {
					log.Panicf("unable to start cache-priming transaction: %s", err)
				}

				// no checks
				defer tx.Rollback(ctx)

				count, bytes, err := cs.selectToCache(
					ctx,
					tx,
					pgCursorStride,
					selector,
				)

				if err != nil {
					log.Panicf("cache-priming via\n%s\nfailed: %s", selector, err)
				}

				atomic.AddInt64(totalCount, count)
				atomic.AddInt64(totalBytes, bytes)

				wg.Done()
			}(sel)
		}

		wg.Wait()

		log.Infof(
			"successfully primed the cache on startup with %s blocks totalling %s bytes: took %.03fs",
			humanize.Comma(*totalCount), humanize.Comma(*totalBytes),
			float64(time.Since(t0).Milliseconds())/1000,
		)
	}()

	return cs, nil
}

func (cs *acs) deploy(ctx context.Context) (err error) {

	// everything is already deployed
	if selErr := cs.dbPool.QueryRow(ctx, `SELECT 42 FROM pg_tables WHERE tablename = 'current'`).Scan(new(int)); selErr == nil {
		return nil
	}

	ddl := []string{
		//
		// basic block storage
		//
		`
		CREATE TABLE IF NOT EXISTS blocks(
			blkid BIGINT NOT NULL PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
			cid BYTEA NOT NULL UNIQUE
		)
		`,

		`
		CREATE TABLE IF NOT EXISTS blocks_content(
			blkid BIGINT NOT NULL PRIMARY KEY REFERENCES blocks( blkid ) DEFERRABLE INITIALLY DEFERRED,
			cid BYTEA NOT NULL REFERENCES blocks( cid ) DEFERRABLE INITIALLY DEFERRED,  -- not indexed, rather just copied over for join-less retrieval
			size INTEGER NOT NULL CONSTRAINT valid_size CHECK ( size > 0 ),
			compressed_content BYTEA,
			linked_blkids BIGINT[] NOT NULL
		)
		`,
		// disable column compression
		`ALTER TABLE blocks_content ALTER COLUMN compressed_content SET STORAGE EXTERNAL`,
		// the GIN would be useful for someone spelunking, but it doesn't do anything for normal ops
		// `CREATE INDEX IF NOT EXISTS blocks_content_reverse_links_gin_idx ON blocks_content USING GIN ( linked_blkids )`,

		`
		CREATE TABLE IF NOT EXISTS blocks_recent(
			blkid BIGINT NOT NULL PRIMARY KEY REFERENCES blocks( blkid ) DEFERRABLE INITIALLY DEFERRED,
			last_access_epoch INTEGER NOT NULL CONSTRAINT valid_last_access CHECK ( last_access_epoch >= 0 )
		)
		`,
		`CREATE INDEX IF NOT EXISTS blocks_recent_last_access_epoch_idx ON blocks_recent ( last_access_epoch )`,

		`
		CREATE OR REPLACE VIEW debug_dangling_cids AS
			SELECT b.cid, b.blkid
				FROM blocks b
				LEFT JOIN blocks_content bc
					USING (blkid)
			WHERE bc.blkid IS NULL
				AND SUBSTRING( b.cid FROM 3 FOR 1 ) != '\x00'              -- it is expected for identity CIDs to have no content
				AND SUBSTRING( b.cid FROM 1 FOR 6 ) != '\x0181e2039220'    -- CommD/P ( baga... )
				AND SUBSTRING( b.cid FROM 1 FOR 7 ) != '\x0182e20381e802'  -- CommR   ( bagb... )
		`,

		//
		// basic chain section
		//
		`
		CREATE TABLE IF NOT EXISTS stateroots(
			blkid BIGINT NOT NULL PRIMARY KEY REFERENCES blocks (blkid) DEFERRABLE INITIALLY DEFERRED,
			message_receipts_blkid BIGINT NOT NULL REFERENCES blocks (blkid) DEFERRABLE INITIALLY DEFERRED,
			epoch INTEGER NOT NULL,
			weight NUMERIC NOT NULL,
			basefee NUMERIC NOT NULL,
			-- FIXME: perhaps add more fields?
			CONSTRAINT no_cycles CHECK ( blkid != message_receipts_blkid )
		)
		`,
		`CREATE INDEX IF NOT EXISTS stateroots_epoch_idx ON stateroots ( epoch )`,
		`CREATE INDEX IF NOT EXISTS stateroots_basefee_idx ON stateroots ( basefee )`,

		`
		CREATE TABLE IF NOT EXISTS chain_headers(
			blkid BIGINT NOT NULL PRIMARY KEY REFERENCES blocks (blkid) DEFERRABLE INITIALLY DEFERRED,
			epoch INTEGER NOT NULL,
			unix_epoch BIGINT NOT NULL,
			messages_blkid BIGINT NOT NULL REFERENCES blocks (blkid) DEFERRABLE INITIALLY DEFERRED,
			miner_actid BIGINT NOT NULL, -- FIXME: refer to the actor table when it exists
			parent_stateroot_blkid BIGINT NOT NULL REFERENCES stateroots (blkid) DEFERRABLE INITIALLY DEFERRED,
			CONSTRAINT no_cycles CHECK ( blkid NOT IN ( parent_stateroot_blkid, messages_blkid ) )
		)
		`,
		`CREATE INDEX IF NOT EXISTS chain_headers_epoch_idx ON chain_headers ( epoch )`,
		`CREATE INDEX IF NOT EXISTS chain_headers_unix_epoch_idx ON chain_headers ( unix_epoch )`,
		`CREATE INDEX IF NOT EXISTS chain_headers_miner_actid_idx ON chain_headers ( miner_actid )`,

		`
		CREATE TABLE IF NOT EXISTS chain_headers_parents(
			header_blkid BIGINT NOT NULL REFERENCES chain_headers (blkid) DEFERRABLE INITIALLY DEFERRED,
			parent_position SMALLINT NOT NULL CONSTRAINT valid_position CHECK ( parent_position >= 0 ),
			parent_blkid BIGINT NOT NULL REFERENCES blocks (blkid) DEFERRABLE INITIALLY DEFERRED, -- FK's on blocks only: we might not have a full chain in-db
			UNIQUE( header_blkid, parent_position ),
			UNIQUE( parent_blkid, header_blkid ),
			CONSTRAINT no_cycles CHECK ( header_blkid != parent_blkid )
		)
		`,

		`
		CREATE TABLE IF NOT EXISTS tipsets(
			tipsetid BIGINT NOT NULL PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
			tipset_key TEXT[] NOT NULL UNIQUE,
			epoch INTEGER NOT NULL,
			wall_time TIMESTAMP WITH TIME ZONE NOT NULL,
			parent_stateroot_blkid BIGINT NOT NULL REFERENCES stateroots (blkid) DEFERRABLE INITIALLY DEFERRED
		)
		`,
		`CREATE INDEX IF NOT EXISTS tipsets_epoch_idx ON tipsets ( epoch )`,
		`CREATE INDEX IF NOT EXISTS tipsets_wall_time_idx ON tipsets ( wall_time )`,

		`
		CREATE TABLE IF NOT EXISTS tipsets_headers(
			tipsetid BIGINT NOT NULL REFERENCES tipsets (tipsetid) DEFERRABLE INITIALLY DEFERRED,
			header_position SMALLINT NOT NULL CONSTRAINT valid_position CHECK ( header_position >= 0 ),
			header_blkid BIGINT NOT NULL REFERENCES chain_headers (blkid) DEFERRABLE INITIALLY DEFERRED,
			UNIQUE( tipsetid, header_position ),
			UNIQUE( header_blkid, tipsetid )
		)
		`,

		`
		DO $$
			BEGIN
				IF NOT EXISTS (SELECT 42 FROM pg_tables WHERE tablename = 'current') THEN
					CREATE TABLE current(
						tipset_key TEXT[] REFERENCES tipsets (tipset_key) DEFERRABLE INITIALLY DEFERRED,
						schema_version SMALLINT NOT NULL CONSTRAINT valid_version CHECK ( schema_version >= 0 ),
						populated BOOL NOT NULL UNIQUE CONSTRAINT single_row_in_table CHECK ( populated IS TRUE )
					);
					INSERT INTO current ( schema_version, populated ) VALUES ( 1, true );
				END IF;
		END $$
		`,

		//
		// ctime/atime tracking
		//
		`
		CREATE TABLE IF NOT EXISTS block_access_log(
			blkid BIGINT NOT NULL, -- No FK to blocks here - we might end up storing the log *before* the blocks have been concurrently committed
			wall_time TIMESTAMP WITH TIME ZONE NOT NULL,
			access_type SMALLINT NOT NULL,
			access_count BIGINT NOT NULL CONSTRAINT access_count_value CHECK ( access_count > 0 ),
			context_epoch INTEGER NOT NULL,
			context_tipsetid BIGINT NOT NULL REFERENCES tipsets (tipsetid) DEFERRABLE INITIALLY DEFERRED
		) PARTITION BY RANGE ( context_epoch )
		`,
		`CREATE INDEX IF NOT EXISTS block_access_log_wall_time_idx ON block_access_log USING BRIN ( wall_time ) WITH ( pages_per_range = 1 )`,
		`CREATE INDEX IF NOT EXISTS block_access_log_context_epoch_idx ON block_access_log USING BRIN ( context_epoch ) WITH ( pages_per_range = 1 )`,

		`CREATE SCHEMA IF NOT EXISTS block_access`,
	}

	// Partition the accesslogs by a week each
	partitionStep := builtin.EpochsInDay * 7

	// Provision 100 weeks to start
	for w := 1; w < 100; w++ {

		ddl = append(ddl, fmt.Sprintf(
			`
			CREATE TABLE IF NOT EXISTS block_access.log_week_%03d
				PARTITION OF block_access_log
				FOR VALUES FROM (%d) TO (%d)
			`,
			w,
			(w-1)*partitionStep, w*partitionStep,
		))
	}

	tx, err := cs.dbPool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return xerrors.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	for _, statement := range ddl {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return xerrors.Errorf("deploy DDL execution failed: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return xerrors.Errorf("failed to finish deploy transaction: %w", err)
	}

	return nil
}

func (cs *acs) Reload() error {
	newPool, err := pgxpool.Connect(context.Background(), cs.dbPool.Config().ConnString())
	if err != nil {
		return err
	}

	oldPool := cs.dbPool
	cs.dbPool = newPool
	oldPool.Close()
	return nil
}

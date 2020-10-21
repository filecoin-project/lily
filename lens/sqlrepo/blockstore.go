package sqlrepo

import (
	"context"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
)

var errNoRows = pgx.ErrNoRows

var dbpool *pgxpool.Pool

var log = logging.Logger("sql")

func DB(connStr string) *pgxpool.Pool {
	if dbpool == nil {
		var err error
		dbpool, err = pgxpool.Connect(context.Background(), connStr)
		if err != nil {
			log.Fatalf("failed to connect to %s: %s", connStr, err)
		}

		for _, ddl := range []string{
			"CREATE TABLE IF NOT EXISTS blocks(" +
				"multiHash TEXT NOT NULL PRIMARY KEY," +
				"initialCodecID INTEGER NOT NULL," +
				"content BYTEA NOT NULL" +
				")",
			"CREATE TABLE IF NOT EXISTS heads(" +
				"seq SERIAL NOT NULL PRIMARY KEY," +
				"ts TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL ," +
				"height BIGINT NOT NULL," +
				"blockCids TEXT NOT NULL" +
				")",
			"CREATE INDEX IF NOT EXISTS height_idx ON heads ( height )",
		} {
			if _, err = dbpool.Exec(context.Background(), ddl); err != nil {
				log.Fatalf("On-connect DDL execution failed: %s", err)
			}
		}
	}

	return dbpool
}

type SqlBlockstore struct {
	db *pgxpool.Pool
}

func NewBlockStore(connstr string) (blockstore.Blockstore, error) {

	sbs := &SqlBlockstore{
		db: DB(connstr),
	}

	// we do not currently use the Identity codec, but just in case...
	return blockstore.WrapIDStore(sbs), nil
}

func keyFromCid(c cid.Cid) (k string) {
	// CB: we can not store complete CIDs as keys - code expects being able
	// to match on multihash alone, likely both over raw *and* CBOR :( :( :(
	//return c.Encode(multibase.MustNewEncoder(multibase.Base64))

	k, _ = multibase.Encode(multibase.Base64, c.Hash())
	return
}

func (sbs *SqlBlockstore) Has(c cid.Cid) (has bool, err error) {
	err = sbs.db.QueryRow(
		context.Background(),
		"SELECT EXISTS( SELECT 42 FROM blocks WHERE multiHash = $1 )",
		keyFromCid(c),
	).Scan(&has)
	return
}

func (sbs *SqlBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {

	retChan := make(chan cid.Cid, 1<<20)

	q, err := sbs.db.Query(
		ctx,
		"SELECT multiHash, initialCodecID FROM blocks",
	)

	if err == errNoRows {
		close(retChan)
		return retChan, nil
	} else if err != nil {
		return nil, err
	}

	go func() {
		defer q.Close()
		defer close(retChan)

		for q.Next() {
			select {

			case <-ctx.Done():
				return

			default:
				var mhEnc string
				var initCodec uint64

				if err := q.Scan(&mhEnc, &initCodec); err == nil {
					if _, mh, err := multibase.Decode(mhEnc); err == nil {
						// CB: It seems we return varying stuff here, depending on store /o\
						// https://github.com/ipfs/go-ipfs-blockstore/blob/v1.0.1/blockstore.go#L229
						retChan <- cid.NewCidV1(initCodec, mh)
						continue
					}
				}

				// if we got that far: we errorred above
				return
			}
		}
	}()

	return retChan, nil
}

func (sbs *SqlBlockstore) Get(c cid.Cid) (blocks.Block, error) {

	var data []byte
	err := sbs.db.QueryRow(
		context.Background(),
		"SELECT content FROM blocks WHERE multiHash = $1",
		keyFromCid(c),
	).Scan(&data)

	switch err {

	case errNoRows:
		return nil, blockstore.ErrNotFound

	case nil:
		return blocks.NewBlockWithCid(data, c)

	default:
		return nil, err

	}
}

func (sbs *SqlBlockstore) GetSize(c cid.Cid) (size int, err error) {

	err = sbs.db.QueryRow(
		context.Background(),
		"SELECT LENGTH(content) FROM blocks WHERE multiHash = $1",
		keyFromCid(c),
	).Scan(&size)

	if err == errNoRows {
		// https://github.com/ipfs/go-ipfs-blockstore/blob/v1.0.1/blockstore.go#L183-L185
		return -1, blockstore.ErrNotFound
	}

	return
}

// Put puts a given block to the underlying datastore
func (sbs *SqlBlockstore) Put(b blocks.Block) (err error) {

	_, err = sbs.db.Exec(
		context.Background(),
		"INSERT INTO blocks( multiHash, initialCodecID, content ) VALUES( $1, $2, $3 ) ON CONFLICT (multiHash) DO NOTHING",
		keyFromCid(b.Cid()),
		b.Cid().Prefix().Codec,
		b.RawData(),
	)

	return
}

func (sbs *SqlBlockstore) PutMany(blks []blocks.Block) error {
	tx, err := sbs.db.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.ReadUncommitted})
	if err != nil {
		return err
	}
	for _, b := range blks {
		if err := sbs.Put(b); err != nil {
			return err
		}
	}
	return tx.Commit(context.Background())
}

func (sbs *SqlBlockstore) getMasterTsKey(ctx context.Context, lookback int) (*types.TipSetKey, error) {

	var headCids string
	if err := sbs.db.QueryRow(
		ctx,
		"SELECT blockcids FROM heads WHERE height = -5 + ( SELECT MAX(height) FROM heads ) ORDER BY seq DESC LIMIT 1",
	).Scan(&headCids); err != nil {
		return nil, err
	}

	cidStrs := strings.Split(headCids, " ")
	cids := make([]cid.Cid, len(cidStrs))
	for _, cs := range cidStrs {
		c, err := cid.Parse(cs)
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	tk := types.NewTipSetKey(cids...)
	return &tk, nil
}

// BEGIN UNIMPLEMENTED

// HashOnRead specifies if every read block should be
// rehashed to make sure it matches its CID.
func (sbs *SqlBlockstore) HashOnRead(enabled bool) {
	log.Warn("HashOnRead toggle not implemented, ignoring")
}

func (sbs *SqlBlockstore) DeleteBlock(cid.Cid) error {
	panic("DeleteBlock not permitted")
}

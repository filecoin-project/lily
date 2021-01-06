package annotated

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/klauspost/compress/zstd"
	"github.com/valyala/gozstd"
	"golang.org/x/xerrors"

	ipfsblock "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	cborgen "github.com/whyrusleeping/cbor-gen"
)

func (cs *acs) dbGet(rootCid cid.Cid, aType accessType) (*blockUnit, error) {

	// decode identity on the spot ( same as blockstore.NewIdStore )
	if rootCid.Prefix().MhType == multihash.IDENTITY {
		bu := &blockUnit{cid: rootCid}

		dmh, err := multihash.Decode(rootCid.Hash())
		if err != nil {
			return nil, err
		}
		bu.hydratedBlock, _ = ipfsblock.NewBlockWithCid(dmh.Digest, rootCid)
		bu.size = uint32(len(dmh.Digest))
		return bu, nil
	}

	// if we can get it out of the cache: do so
	if resp, found := cs.cache.Get(rootCid.Bytes()); found {
		bu := resp.(*blockUnit)
		cs.noteAccess(*bu.dbID, time.Now(), aType|PREEXISTING)
		return bu, nil
	}

	var bu blockUnit
	var cidBytes, compressedContent []byte
	err := cs.dbPool.QueryRow(
		context.Background(),
		`
		SELECT b.blkid, b.cid, bc.size, bc.compressed_content
			FROM blocks_content bc
			JOIN blocks b
				ON
					bc.blkid = b.blkid
						AND
					b.cid = $1::BYTEA
		`,
		rootCid.Bytes(),
	).Scan(&bu.dbID, &cidBytes, &bu.size, &compressedContent)

	if err == pgx.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if _, bu.cid, err = cid.CidFromBytes(cidBytes); err != nil {
		return nil, err
	}

	if len(compressedContent) == 0 {
		return nil, fmt.Errorf("unexpected zero-length content %#v for cid %s", compressedContent, bu.cid.String())
	}

	// we are good - copy, set and log

	// must copy: https://github.com/jackc/pgx/issues/845#issuecomment-705550012
	bu.compressedContent = append(
		make([]byte, 0, len(compressedContent)),
		compressedContent...,
	)
	cs.cache.Set(cidBytes, &bu, int64(bu.size))
	cs.noteAccess(*bu.dbID, time.Now(), aType)

	return &bu, nil
}

func (cs *acs) dbPut(blks []ipfsblock.Block) (err error) {

	if len(blks) == 0 {
		return nil
	}

	var wgCompress, wgRecurse sync.WaitGroup

	referencedCids := newCidSet()
	toInsertUnits := make(map[cid.Cid]*blockUnit, len(blks))

	// use the same PUT time for all members in same batch
	now := time.Now()

	// determine what do we want to insert, and asynchronously start:
	// - precompression
	// - link parsing
	var linkCount int
	for i := range blks {
		blkCid := blks[i].Cid()

		// we were given duplicate CIDs to insert :(
		if _, isDuplicate := toInsertUnits[blkCid]; isDuplicate {
			log.Debugf("unexpected duplicate insert of block %s", blkCid)
			continue
		}

		// if it is in the cache - it's already fully processed, there's no other way
		if cbu, isCached := cs.cache.Get(blkCid.Bytes()); isCached {
			cs.noteAccess(*(cbu.(*blockUnit)).dbID, now, PUT|PREEXISTING)
			continue
		}

		bu := &blockUnit{
			cid:           blkCid,
			hydratedBlock: blks[i],
			size:          uint32(len(blks[i].RawData())),
		}
		toInsertUnits[blkCid] = bu
		referencedCids.Add(blkCid)

		// prepare the links unless Raw
		if blkCid.Prefix().Codec != cid.Raw {

			wgRecurse.Add(1)
			cs.limiterBlockParse <- struct{}{}

			go func() {

				seen := cid.NewSet()
				if err := cborgen.ScanForLinks(bytes.NewReader(bu.hydratedBlock.RawData()), func(c cid.Cid) {
					if seen.Visit(c) {
						referencedCids.Add(c)
						bu.parsedLinks = append(bu.parsedLinks, c)
						linkCount++
					}
				}); err != nil {
					bu.errHolder = xerrors.Errorf("cborgen.ScanForLinks: %w", err)
				}

				<-cs.limiterBlockParse
				wgRecurse.Done()
			}()
		}

		// no content to insert for identity cids: no compression
		if blkCid.Prefix().MhType == multihash.IDENTITY {
			continue
		}

		wgCompress.Add(1)
		cs.limiterCompress <- struct{}{}

		go func() {

			// Alternative way of compressing based on 	"github.com/klauspost/compress/zstd"
			// ( slightly worse results than gozstd, but might be preferrable avoiding cgo switching  )
			//
			// zstdEnc, _ = zstd.NewWriter(nil,
			// 	zstd.WithNoEntropyCompression(false),
			// 	zstd.WithAllLitEntropyCompression(true),
			// 	zstd.WithSingleSegment(true),
			// 	zstd.WithEncoderCRC(false),
			// 	zstd.WithEncoderLevel(zstd.SpeedBestCompression),
			// )
			// ...
			// bu.compressedContent = zstdEnc.EncodeAll(
			// 	bu.block.RawData(),
			// 	make([]byte, 0, bu.size),
			// )

			bu.compressedContent = gozstd.CompressLevel(
				make([]byte, 0, bu.size),
				bu.hydratedBlock.RawData(),
				zstdCompressLevel,
			)

			<-cs.limiterCompress
			wgCompress.Done()
		}()
	}

	// everything happened to be in the cache
	if len(toInsertUnits) == 0 {
		return nil
	}

	// wait for all linkparses to finish
	wgRecurse.Wait()

	// if any of the linkparses failed - stop
	for _, bu := range toInsertUnits {
		if bu.errHolder != nil {
			return bu.errHolder
		}
	}

	ctx := context.TODO()
	tx, err := cs.dbPool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadUncommitted}) // ReadUncommitted is unsupported but eh...
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			log.Errorf("UNEXPECTED rollback while storing blocks: %s", err)
			tx.Rollback(ctx)
		} else {

			// Everything worked: call it
			err = tx.Commit(ctx)

			// Everything got inserted: populate the cache
			if err == nil {
				for _, bu := range toInsertUnits {
					cs.cache.Set(bu.cid.Bytes(), bu, int64(bu.size))
				}
			}
		}
	}()

	dbIDs, err := cs.mapCids(ctx, tx, referencedCids)
	if err != nil {
		return err
	}

	// populate in-struct db-side-id, increment access counters
	//
	for blkCid := range toInsertUnits {
		bu := toInsertUnits[blkCid]

		bu.dbID = dbIDs[string(bu.cid.Bytes())].dbID

		if dbIDs[string(bu.cid.Bytes())].contentFound {
			cs.noteAccess(*bu.dbID, now, PUT|PREEXISTING)
		} else {
			cs.noteAccess(*bu.dbID, now, PUT)
		}
	}

	// all compression activities should be done by now
	//
	wgCompress.Wait()

	// Figure out what content are we inserting
	//
	contentEntries := make([][]interface{}, 0, len(toInsertUnits))
	for blkCid, bu := range toInsertUnits {

		// Content row already exists: skip content dance
		if dbIDs[string(blkCid.Bytes())].contentFound {
			continue
		}

		linkIDs := make([]uint64, 0, len(bu.parsedLinks))
		for _, c := range bu.parsedLinks {
			linkIDs = append(linkIDs, *dbIDs[string(c.Bytes())].dbID)
		}

		// FIXME - minimal sancheck, remove at some point
		if blkCid.Prefix().Codec == cid.Raw && len(linkIDs) != 0 {
			log.Panicf("impossibly %d links parsed out of raw block %s", len(linkIDs), blkCid.String())
		}
		if bu.compressedContent != nil && len(bu.compressedContent) == 0 {
			log.Panicf("invalid 0-length compressed content for CID %s", blkCid.String())
		}
		if (blkCid.Prefix().MhType == multihash.IDENTITY) != (bu.compressedContent == nil) {
			log.Panicf(
				"impossibly isIdentity(%t) != contentIsNULL(%t) for %d bytes of CID %s",
				(blkCid.Prefix().MhType == multihash.IDENTITY),
				(bu.compressedContent == nil),
				len(bu.compressedContent),
				blkCid.String(),
			)
		}

		contentEntries = append(contentEntries, []interface{}{
			bu.dbID, blkCid.Bytes(), bu.size, bu.compressedContent, linkIDs,
		})
	}

	// If nothing: we are done
	//
	if len(contentEntries) == 0 {
		return nil
	}

	// Insert the complete content row and be done
	//
	return rapidPopulate(
		ctx,
		tx,
		"blocks_content",
		[]string{"blkid", "cid", "size", "compressed_content", "linked_blkids"},
		contentEntries,
	)
}

// populate through a temptable to allow ON CONFLICT ... DO NOTHING to work
func rapidPopulate(ctx context.Context, tx pgx.Tx, intoTable string, columns []string, data [][]interface{}) error {

	randName := "tmptable_" + randBytesAsHex()

	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf(
			`CREATE TEMPORARY TABLE %s ( LIKE %s ) ON COMMIT DROP`,
			randName,
			intoTable,
		),
	); err != nil {
		return err
	}

	if _, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{randName},
		columns,
		pgx.CopyFromRows(data),
	); err != nil {
		return err
	}

	columnList := strings.Join(columns, ",")

	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf(
			`
			INSERT INTO %s ( %s )
				SELECT %s FROM %s ORDER BY 1
				ON CONFLICT DO NOTHING
			`,
			intoTable, columnList,
			columnList, randName,
		),
	); err != nil {
		return err
	}

	return nil
}

type idUnit struct {
	dbID         *uint64
	contentFound bool
}

// FIXME - this is the heaviest call: there got to be a way to do things 10x as fast
func (cs *acs) mapCids(ctx context.Context, tx pgx.Tx, initialCids *cidSet) (map[string]idUnit, error) {

	finalCidMap := make(map[string]idUnit, initialCids.Len())

	if initialCids.Len() == 0 {
		return finalCidMap, nil
	}

	// pull anything we know from the cache, otherwise we have to go to the db and map
	remainingToMap := make([][]byte, 0, initialCids.Len())
	for _, c := range initialCids.Keys() {
		cidBytes := c.Bytes()
		if resp, found := cs.cache.Get(cidBytes); found {
			bu := resp.(*blockUnit)
			finalCidMap[string(cidBytes)] = idUnit{dbID: bu.dbID, contentFound: true}
		} else {
			remainingToMap = append(remainingToMap, cidBytes)
		}
	}

	// perhaps everything was cached
	if len(remainingToMap) == 0 {
		return finalCidMap, nil
	}

	// Counterintuitively - first check what's available ( seems faster that way )
	//
	// This used to be a more elaborate INSERT+SELECT cte, but it doesn't work very well:
	//
	// 	WITH
	// 		new_blkids AS (
	// 			INSERT INTO blocks ( cid ) SELECT UNNEST( $1::BYTEA[] )
	// 				ON CONFLICT DO NOTHING
	// 			RETURNING cid, blkid
	// 		)
	// 	SELECT cid, blkid, false FROM new_blkids
	// UNION
	// 	SELECT
	// 		b.cid, b.blkid, EXISTS (
	// 			SELECT 42
	// 				FROM blocks_content bc
	// 			WHERE bc.blkid = b.blkid
	// 		)
	// 		FROM blocks b
	//
	// Read `Concurrency issue 1` here to understand why
	// https://stackoverflow.com/a/42217872
	//
	existingIDs, err := tx.Query(
		ctx,
		`
		SELECT
			b.blkid,
			b.cid,
			EXISTS ( SELECT 42 FROM blocks_content bc WHERE bc.blkid = b.blkid )
		FROM blocks b
		WHERE cid = ANY( $1::BYTEA[] )
		`,
		remainingToMap,
	)
	if err != nil {
		return nil, err
	}

	for existingIDs.Next() {
		var (
			dbID         *uint64
			cidBytes     []byte
			contentFound bool
		)
		if err := existingIDs.Scan(&dbID, &cidBytes, &contentFound); err != nil {
			return nil, err
		}
		finalCidMap[string(cidBytes)] = idUnit{dbID: dbID, contentFound: contentFound}
	}
	if err = existingIDs.Err(); err != nil {
		return nil, err
	}

	// see what remains to insert
	remainingToMap = remainingToMap[:0]
	for _, c := range initialCids.Keys() {
		if _, known := finalCidMap[string(c.Bytes())]; !known {
			remainingToMap = append(remainingToMap, c.Bytes())
		}
	}

	// oh neat: we are done
	if len(remainingToMap) == 0 {
		return finalCidMap, nil
	}

	// pre-sort-ed keys in a bid to avoid deadlocks during concurrent index updates
	sort.Slice(remainingToMap, func(i, j int) bool {
		return bytes.Compare(remainingToMap[i], remainingToMap[j]) < 0
	})

	newIDs, err := tx.Query(
		ctx,
		`
		INSERT INTO blocks ( cid ) SELECT UNNEST( $1::BYTEA[] )
			ON CONFLICT DO NOTHING
		RETURNING blkid, cid
		`,
		remainingToMap,
	)
	if err != nil {
		return nil, err
	}

	for newIDs.Next() {
		var (
			dbID     *uint64
			cidBytes []byte
		)
		if err = newIDs.Scan(&dbID, &cidBytes); err != nil {
			return nil, err
		}
		finalCidMap[string(cidBytes)] = idUnit{dbID: dbID}
	}
	if err = newIDs.Err(); err != nil {
		return nil, err
	}

	// because of concurrency we might have dropped a few IDs - one last query to mop up
	remainingToMap = remainingToMap[:0]
	for _, c := range initialCids.Keys() {
		if _, known := finalCidMap[string(c.Bytes())]; !known {
			remainingToMap = append(remainingToMap, c.Bytes())
		}
	}

	// ok, now indeed done
	if len(remainingToMap) == 0 {
		return finalCidMap, nil
	}

	// damn... we are not
	concurrentInserts, err := tx.Query(
		ctx,
		`
		SELECT
			b.blkid,
			b.cid,
			EXISTS ( SELECT 42 FROM blocks_content bc WHERE bc.blkid = b.blkid )
		FROM blocks b
		WHERE cid = ANY( $1::BYTEA[] )
		`,
		remainingToMap,
	)
	if err != nil {
		return nil, err
	}

	for concurrentInserts.Next() {
		var (
			dbID         *uint64
			cidBytes     []byte
			contentFound bool
		)
		if err := concurrentInserts.Scan(&dbID, &cidBytes, &contentFound); err != nil {
			return nil, err
		}
		finalCidMap[string(cidBytes)] = idUnit{dbID: dbID, contentFound: contentFound}
	}
	if err = concurrentInserts.Err(); err != nil {
		return nil, err
	}

	// if we got that far - might as well validate it all
	var missed []string
	for _, c := range initialCids.Keys() {
		if finalCidMap[string(c.Bytes())].dbID == nil {
			missed = append(missed, c.String())
		}
	}
	if len(missed) > 0 {
		log.Panicf(
			"a number of CIDs were not successfully mapped in the central table:\n%s",
			strings.Join(missed, "\n"),
		)
	}

	return finalCidMap, nil
}

//
// Decompressor
var zstDec, _ = zstd.NewReader(nil)

func (bu *blockUnit) block() (ipfsblock.Block, error) {

	bu.mu.Lock()
	defer bu.mu.Unlock()

	if bu.errHolder != nil {
		return nil, bu.errHolder
	}

	if bu.hydratedBlock == nil {

		var blkContent []byte
		blkContent, bu.errHolder = zstDec.DecodeAll(
			bu.compressedContent,
			make([]byte, 0, bu.size),
		)
		if bu.errHolder != nil {
			return nil, bu.errHolder
		}

		// FIXME: for now always validate the blocks, just to be safe
		{
			var recalcCid cid.Cid
			recalcCid, bu.errHolder = bu.cid.Prefix().Sum(blkContent)
			if bu.errHolder != nil {
				return nil, bu.errHolder
			}
			if !recalcCid.Equals(bu.cid) {
				bu.errHolder = fmt.Errorf("Hash Mismatch")
				return nil, bu.errHolder
			}
		}

		bu.hydratedBlock, bu.errHolder = ipfsblock.NewBlockWithCid(blkContent, bu.cid)
		if bu.errHolder != nil {
			return nil, bu.errHolder
		}

		// we can GC this now
		bu.compressedContent = nil
	}

	return bu.hydratedBlock, bu.errHolder
}

//
// misc stuffz
//
const randBytesCount = 16

func randBytesAsHex() string {
	randBinName := make([]byte, randBytesCount)
	rand.Read(randBinName)
	return fmt.Sprintf("%x", randBinName)
}

//
// kludge-y concurrency-safe cid.Set
//
type cidSet struct {
	mu  sync.Mutex
	set map[cid.Cid]struct{}
}

func newCidSet() *cidSet {
	cs := new(cidSet)
	cs.set = make(map[cid.Cid]struct{}, 64)
	return cs
}

func (cs *cidSet) Add(c cid.Cid) {
	cs.mu.Lock()
	cs.set[c] = struct{}{}
	cs.mu.Unlock()
}

func (cs *cidSet) Len() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.set)
}

func (cs *cidSet) Keys() []cid.Cid {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	ret := make([]cid.Cid, 0, len(cs.set))

	for c := range cs.set {
		ret = append(ret, c)
	}

	return ret
}

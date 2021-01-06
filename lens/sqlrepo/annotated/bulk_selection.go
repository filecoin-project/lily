package annotated

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/jackc/pgx/v4"
	"github.com/multiformats/go-multihash"
	"golang.org/x/xerrors"

	"github.com/ipfs/go-cid"
)

func (cs *acs) selectToCache(ctx context.Context, tx pgx.Tx, cursorStride uint16, sqlSelect string, sqlArgs ...interface{}) (blockCount, bytes int64, err error) {

	cursorName := `cursor_` + randBytesAsHex()

	// https://www.postgresql.org/docs/12/sql-declare.html
	_, err = tx.Exec(
		ctx,
		fmt.Sprintf(
			"DECLARE %s INSENSITIVE BINARY NO SCROLL CURSOR WITHOUT HOLD\nFOR\n%s",
			cursorName,
			sqlSelect,
		),
		sqlArgs...,
	)

	if err != nil {
		err = xerrors.Errorf("cursor declaration failed: %w", err)
		return
	}

	cacheQueue := make(chan *blockUnit, cursorStride*2)
	cachingDone := make(chan struct{})

	go func() {
		for {
			bu, chanIsOpen := <-cacheQueue
			if !chanIsOpen {
				close(cachingDone)
				return
			}
			cs.cache.Set(bu.cid.Bytes(), bu, int64(bu.size))
			bytes += int64(bu.size)
			blockCount++
		}
	}()

	defer func() {
		close(cacheQueue)
		<-cachingDone
	}()

	for {

		var rows pgx.Rows
		rows, err = tx.Query(
			ctx, fmt.Sprintf(
				`FETCH FORWARD %d FROM %s`,
				cursorStride,
				cursorName,
			))

		if err != nil {
			return
		}

		var resultsSeen bool

		for rows.Next() {

			resultsSeen = true

			var bu blockUnit
			var cidBytes, compressedContent []byte

			if err = rows.Scan(&bu.dbID, &cidBytes, &bu.size, &compressedContent); err != nil {
				return
			}
			if _, bu.cid, err = cid.CidFromBytes(cidBytes); err != nil {
				return
			}

			// a recursive query might return an identity CID, which will have no content
			if bu.cid.Prefix().MhType == multihash.IDENTITY {
				continue
			}

			if len(compressedContent) == 0 {
				err = fmt.Errorf("unexpected zero-length content %#v for cid %s", compressedContent, bu.cid.String())
				return
			}

			// must copy: https://github.com/jackc/pgx/issues/845#issuecomment-705550012
			bu.compressedContent = append(
				make([]byte, 0, len(compressedContent)),
				compressedContent...,
			)

			cacheQueue <- &bu
		}

		if rows.Err() != nil {
			err = rows.Err()
			return
		}

		if !resultsSeen {
			break
		}
	}

	return
}

func (cs *acs) selectRangeBlocksToCache(ctx context.Context, tx pgx.Tx, minEpoch, maxEpoch abi.ChainEpoch, cursorStride uint16) (blockCount, bytes int64, err error) {

	return cs.selectToCache(
		ctx,
		tx,
		cursorStride,
		`
		WITH RECURSIVE
			cte_roots( blkid ) AS (

					SELECT blkid
						FROM stateroots
					WHERE	epoch BETWEEN $1 AND $2

				UNION

					SELECT blkid
						FROM chain_headers
					WHERE epoch BETWEEN $1 AND $2
			),

			cte_dag( blkid ) AS (

					SELECT blkid FROM cte_roots

				UNION

					SELECT UNNEST( bc.linked_blkids )
						FROM cte_dag
						JOIN blocks_content bc
							USING( blkid )
			)

		SELECT bc.blkid, bc.cid, bc.size, bc.compressed_content
			FROM blocks_content bc
		WHERE
			bc.blkid IN ( SELECT blkid FROM cte_dag )
				AND
			bc.compressed_content IS NOT NULL
		`,
		minEpoch,
		maxEpoch,
	)
}

func (cs *acs) selectGraphToCache(ctx context.Context, tx pgx.Tx, roots *cidSet, maxDepth uint8, cursorStride uint16) (blockCount, bytes int64, err error) {

	if roots == nil ||
		roots.Len() == 0 {
		return
	}

	rootCidBytes := make([][]byte, 0, roots.Len())
	for _, c := range roots.Keys() {
		rootCidBytes = append(rootCidBytes, c.Bytes())
	}

	return cs.selectToCache(
		ctx,
		tx,
		cursorStride,
		// FIXME - suboptimal, needs to be rewritten with an annotation-ignoring UNION
		`
		WITH RECURSIVE
			cte_dag( blkid, level ) AS (
					SELECT blkid, 0
						FROM blocks
					WHERE cid = ANY( $1::BYTEA[] )
				UNION
					SELECT UNNEST( bc.linked_blkids ), cte_dag.level+1
						FROM blocks_content bc
						JOIN cte_dag
							ON bc.blkid = cte_dag.blkid AND cte_dag.level < $2
			)
		SELECT bc.blkid, bc.cid, bc.size, bc.compressed_content
			FROM blocks_content bc
		WHERE
			bc.blkid IN ( SELECT blkid FROM cte_dag )
				AND
			bc.compressed_content IS NOT NULL
		`,
		rootCidBytes,
		maxDepth, // 1-based: each integer means one extra layer of children
	)
}

package annotated

import (
	"context"
	"sort"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/jackc/pgx/v4"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/multiformats/go-varint"
)

func (cs *acs) noteAccess(dbID uint64, t time.Time, atype accessType) {
	// not used since this is a read-only instance.
	return
	/*
		cs.mu.Lock()

		// FIXME: not sure what to track s "recent" exactly: just write down all non-PUT's for now...
		if atype&MASKTYPE != PUT {
			cs.accessStatsRecent[dbID] = struct{}{}
		}

		if cs.accessStatsHiRes != nil {
			cs.accessStatsHiRes[accessUnit{
				atUnix:     t.Truncate(time.Millisecond),
				dbID:       dbID,
				accessType: atype,
			}]++
		}

		cs.mu.Unlock()
	*/
}

func (cs *acs) SetCurrentTipset(ctx context.Context, ts *types.TipSet) (didChange bool, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// protect from double-sets
	if cs.currentTipset != nil && cs.currentTipset.Equals(ts) {
		return false, nil
	}

	headers := ts.Blocks()
	if len(headers) == 0 {
		// null round - counters persist until a real tipset
		return false, nil
	}

	// FIXME - perhaps remove this coherence sanity check
	for i, hdr := range headers {
		if i == 0 {
			continue
		}

		switch {

		case hdr.ParentBaseFee.String() != headers[0].ParentBaseFee.String():
			return false, xerrors.Errorf("unexpected BaseFee mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
				hdr.Cid().String(), i, hdr.ParentBaseFee.String(),
				headers[0].Cid().String(), headers[0].ParentBaseFee.String(),
			)

		case hdr.ParentWeight.String() != headers[0].ParentWeight.String():
			return false, xerrors.Errorf("unexpected Weight mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
				hdr.Cid().String(), i, hdr.ParentWeight.String(),
				headers[0].Cid().String(), headers[0].ParentWeight.String(),
			)

		case hdr.ParentStateRoot.String() != headers[0].ParentStateRoot.String():
			return false, xerrors.Errorf("unexpected ParentStateroot mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
				hdr.Cid().String(), i, hdr.ParentStateRoot.String(),
				headers[0].Cid().String(), headers[0].ParentStateRoot.String(),
			)

		case hdr.ParentMessageReceipts.String() != headers[0].ParentMessageReceipts.String():
			return false, xerrors.Errorf("unexpected ParentMessageReceipts mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
				hdr.Cid().String(), i, hdr.ParentMessageReceipts.String(),
				headers[0].Cid().String(), headers[0].ParentMessageReceipts.String(),
			)
		}
	}

	tx, err := cs.dbPool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			log.Errorf("UNEXPECTED rollback during tipset bookkeeping: %s", err)
			didChange = false
			tx.Rollback(ctx)
		}
	}()

	// First collect all cids we care about
	cidsToMap := newCidSet()
	tskArray := make([]string, 0, len(headers))
	for _, hdr := range headers {
		tskArray = append(tskArray, hdr.Cid().String())
		cidsToMap.Add(hdr.Cid())
		cidsToMap.Add(hdr.Messages)
		for _, p := range hdr.Parents {
			cidsToMap.Add(p)
		}
	}
	cidsToMap.Add(ts.ParentState())
	cidsToMap.Add(headers[0].ParentMessageReceipts)

	// grab dbIDs for all of the interesting CIDs
	blockDbIDs, err := cs.mapCids(ctx, tx, cidsToMap)
	if err != nil {
		return false, err
	}

	var tsDbID uint64
	var preExisting bool
	if err = tx.QueryRow(
		ctx,
		`
		WITH new_tipset AS (
			INSERT INTO tipsets ( tipset_key, epoch, wall_time, parent_stateroot_blkid )
				VALUES ( $1::TEXT[], $2, $3, $4 )
			ON CONFLICT DO NOTHING
			RETURNING tipsetid
		)
		SELECT tipsetid, false FROM new_tipset
	UNION
		SELECT tipsetid, true FROM tipsets WHERE tipset_key = $1::TEXT[]
		`,
		tskArray,
		ts.Height(),
		time.Now().Truncate(time.Millisecond),
		blockDbIDs[string(ts.ParentState().Bytes())].dbID,
	).Scan(&tsDbID, &preExisting); err != nil {
		// unlike the block-insertion codepath here we hold an exclusive lock: an error is an error
		err = xerrors.Errorf("unexpectedly failed to retrieve just-inserted tipset id: %w", err)
		return false, err
	}

	pgBatch := &pgx.Batch{}

	if !preExisting {

		pgBatch.Queue(
			`
			INSERT INTO stateroots ( blkid, message_receipts_blkid, epoch, weight, basefee )
				VALUES ( $1, $2, $3, $4, $5 )
			ON CONFLICT DO NOTHING
			`,
			blockDbIDs[string(headers[0].ParentStateRoot.Bytes())].dbID,
			blockDbIDs[string(headers[0].ParentMessageReceipts.Bytes())].dbID,
			headers[0].Height,
			headers[0].ParentWeight.String(),
			headers[0].ParentBaseFee.String(),
		)

		for i, hdr := range headers {

			if hdr.Miner.Protocol() != address.ID {
				return false, xerrors.Errorf("unexpected miner address '%s' in chain block %s", hdr.Miner, hdr.Cid())
			}
			minerid, _, err := varint.FromUvarint(hdr.Miner.Payload())
			if err != nil {
				return false, xerrors.Errorf("failed to parse address '%s' payload: %w", hdr.Miner, err)
			}

			hCidStr := string(hdr.Cid().Bytes())

			pgBatch.Queue(
				`
				INSERT INTO tipsets_headers ( tipsetid, header_position, header_blkid )
					VALUES ( $1, $2, $3 )
				ON CONFLICT DO NOTHING
				`,
				tsDbID,
				i,
				blockDbIDs[hCidStr].dbID,
			)

			pgBatch.Queue(
				`
				INSERT INTO chain_headers (
					blkid,
					epoch, unix_epoch, miner_actid,
					messages_blkid,
					parent_stateroot_blkid
				) VALUES (
					$1,
					$2, $3, $4,
					$5,
					$6
				)
				ON CONFLICT DO NOTHING
				`,
				blockDbIDs[hCidStr].dbID,
				hdr.Height, hdr.Timestamp, minerid,
				blockDbIDs[string(hdr.Messages.Bytes())].dbID,
				blockDbIDs[string(hdr.ParentStateRoot.Bytes())].dbID,
			)

			for i, hpCid := range hdr.Parents {
				pgBatch.Queue(
					`
					INSERT INTO chain_headers_parents ( header_blkid, parent_position, parent_blkid )
						VALUES ( $1, $2, $3 )
					ON CONFLICT DO NOTHING
					`,
					blockDbIDs[hCidStr].dbID,
					i,
					blockDbIDs[string(hpCid.Bytes())].dbID,
				)
			}
		}
	}

	// always move the current, duplicate or not
	pgBatch.Queue(
		`UPDATE current SET tipset_key = $1`,
		tskArray,
	)

	// execute entire batch in one go
	if err = tx.SendBatch(
		ctx,
		pgBatch,
	).Close(); err != nil {
		return false, err
	}

	// we got that far: let's try to commit
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}

	// it all worked! re-assign tipset, keep track of amount of exact +1 height jumps
	if cs.currentTipset != nil && (ts.Height()-cs.currentTipset.Height()) == 1 {
		// if delta is exactly +1 we are no longer jumping around on startup
		cs.linearSyncEventCount++
	}
	cs.currentTipset = ts

	// save the individual access logs asynchronously, but panic on error
	if cs.accessStatsHiRes != nil {

		hiResLogs := make([][]interface{}, 0, len(cs.accessStatsHiRes))
		for au, count := range cs.accessStatsHiRes {
			hiResLogs = append(hiResLogs, []interface{}{
				au.atUnix, au.dbID, au.accessType, count, ts.Height(), tsDbID,
			})
		}

		// reset the counters: if the insert fails - it fails
		cs.accessStatsHiRes = make(map[accessUnit]uint64, 16384)

		// launch into background, panic if fails
		go func() {
			if _, err := cs.dbPool.CopyFrom(
				context.Background(),
				pgx.Identifier{"block_access_log"},
				[]string{"wall_time", "blkid", "access_type", "access_count", "context_epoch", "context_tipsetid"},
				pgx.CopyFromRows(hiResLogs),
			); err != nil {
				log.Panicf("failure writing high-resolution access logs for tipset %s: %s", ts.String(), err)
			}
		}()
	}

	// if the previous "last-get" update is done - do the next one ( see default: at end )
	// we do it asynchronously in a one-at-a-time, fire-and-forget manner, which is good enough here
	select {
	case cs.limiterSetLastAccess <- struct{}{}:

		idsToUpdate := make([]uint64, 0, len(cs.accessStatsRecent))
		for dbID := range cs.accessStatsRecent {
			idsToUpdate = append(idsToUpdate, dbID)
		}

		// now that we decided what to update: reset the counter
		// if the update further down fails - it fails
		cs.accessStatsRecent = make(map[uint64]struct{}, 16384)

		// somehow nothing to do...
		if len(idsToUpdate) == 0 {

			<-cs.limiterSetLastAccess

		} else {

			go func() (err error) {
				defer func() { <-cs.limiterSetLastAccess }()

				sort.Slice(idsToUpdate, func(i, j int) bool {
					return idsToUpdate[i] < idsToUpdate[j]
				})

				ctx := context.Background()

				tx, err := cs.dbPool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
				if err != nil {
					return
				}
				defer func() {
					if err == nil {
						err = tx.Commit(ctx)
					}
					if err != nil {
						log.Errorf("UNEXPECTED rollback during recent-access bookkeeping for tipset %s: %s", ts.String(), err)
						tx.Rollback(ctx)
					}
				}()

				pgBatch := &pgx.Batch{}

				// we stabilized sufficiently: it is safe to start purging logs
				if cs.linearSyncEventCount > trackRecentTipsets {
					// Delete everything that is:
					// - beyond trackRecentTipsets in the past
					// - more than 3 days in the future
					pgBatch.Queue(
						`
						DELETE FROM blocks_recent
						WHERE last_access_epoch NOT BETWEEN $1 AND $2
						`,
						ts.Height()-trackRecentTipsets, (ts.Height() + 3*builtin.EpochsInDay),
					)
				}

				// add new logentries
				pgBatch.Queue(
					`
					INSERT INTO blocks_recent( blkid, last_access_epoch )
						SELECT UNNEST( $1::BIGINT[] ), $2
					ON CONFLICT ( blkid ) DO
						UPDATE SET last_access_epoch = $2
					`,
					idsToUpdate,
					ts.Height(),
				)

				return tx.SendBatch(ctx, pgBatch).Close()
			}()
		}

	default:
		// nothing - wait for the next round
	}

	return true, nil
}

func (cs *acs) GetCurrentTipset(ctx context.Context) []cid.Cid {
	row := cs.dbPool.QueryRow(ctx, "SELECT tipset_key FROM current")
	rawcids := make([]string, 0)
	if err := row.Scan(&rawcids); err != nil {
		log.Errorf("UNEXPECTED empty result when getting tipset key", err)
		return nil
	}
	cids := make([]cid.Cid, 0, len(rawcids))
	for _, c := range rawcids {
		cx, err := cid.Parse(c)
		if err != nil {
			log.Errorf("UNEXPECTED unparsable cid when getting tipset key", err)
			continue
		}
		cids = append(cids, cx)
	}
	return cids
}

package tstracker

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	"github.com/jackc/pgx/v4"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	pgchainbs "github.com/filecoin-project/go-bs-postgres-chainnotated"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

func (tcs *tcs) CurrentDbTipSetKey(ctx context.Context) (*types.TipSetKey, abi.ChainEpoch, error) {

	tsks, epoch, err := tcs.CurrentFilTipSetKey(ctx)
	if err != nil {
		return nil, -1, err
	}

	tsk := types.NewTipSetKey(tsks...)

	return &tsk, epoch, nil
}

func (tcs *tcs) GetCurrentTipset(ctx context.Context, lookback int) (*types.TipSetKey, error) {
	tsks, _, err := tcs.CurrentFilTipSetKey(ctx)
	if err != nil {
		return nil, err
	}
	tsk := types.NewTipSetKey(tsks...)

	return &tsk, nil
}

func (tcs *tcs) StoreTipSetVist(ctx context.Context, ts *types.TipSet, isHeadChange bool) error {

	// note the time before anything else
	visitTime := time.Now()

	if !tcs.IsWritable() {
		return xerrors.New("unable to StoreTipSetVist on read-only chainstore")
	}
	if tcs.InstanceNamespace() == "" {
		return xerrors.New("unable to StoreTipSetVist without a specified instance namespace")
	}

	headers := ts.Blocks()
	// somehow got called with a null round - nothing to record
	if len(headers) == 0 {
		return nil
	}

	// if a headchange: take an early lock - this allows us to properly serialize
	// events from competing workers
	if isHeadChange {
		tcs.lastSeenTipSetMu.Lock()
		defer tcs.lastSeenTipSetMu.Unlock()

	} else {
		tcs.lastSeenTipSetMu.RLock()
		lastTs := tcs.lastSeenTipSet
		tcs.lastSeenTipSetMu.RUnlock()

		// make no records for duplicate visits ( they do happen inexplicably )
		if lastTs != nil && lastTs.Equals(ts) {
			return nil
		}
	}

	tskCidsBytes := make([][]byte, 0, len(headers))
	for _, h := range headers {
		tskCidsBytes = append(tskCidsBytes, h.Cid().Bytes())
	}

	// check for the TS record: can skip destructuring (which isn't cheap)
	var tsDbOrdinal *int32
	err := tcs.PgxPool().QueryRow(
		ctx,
		`SELECT tipset_ordinal FROM fil_common_base.tipsets WHERE tipset_cids = $1::BYTEA[]`,
		tskCidsBytes,
	).Scan(&tsDbOrdinal)

	if err == pgx.ErrNoRows { // no rows == new, not-yet-stored tipset
		var tsd *pgchainbs.DestructuredFilTipSetData
		tsd, err = destructureTipset(ts)
		if err != nil {
			return xerrors.Errorf("failed unrolling metadata for tipset %s (%d): %w",
				ts.Key().String(),
				ts.Height(),
				err,
			)
		}
		tsDbOrdinal, err = tcs.StoreFilTipSetData(ctx, tsd)
		if err != nil {
			return xerrors.Errorf("failed storing unrolled metadata for tipset %s (%d): %w",
				ts.Key().String(),
				ts.Height(),
				err,
			)
		}
	} else if err != nil {
		return err
	}

	err = tcs.StoreFilTipSetVisit(ctx, tsDbOrdinal, ts.Height(), visitTime, isHeadChange)
	if err != nil {
		return err
	}

	// it all worked! do final access housekeeping

	if !isHeadChange {
		// if we were changing head: we already locked the entire scope earlier
		tcs.lastSeenTipSetMu.Lock()
		defer tcs.lastSeenTipSetMu.Unlock()
	}

	tcs.lastSeenTipSet = ts

	if isHeadChange {
		return tcs.FlushAccessLogs(ts.Height(), tsDbOrdinal)
	}

	return nil
}

func destructureTipset(ts *types.TipSet) (*pgchainbs.DestructuredFilTipSetData, error) {

	headers := ts.Blocks()

	tsd := &pgchainbs.DestructuredFilTipSetData{
		Epoch:                    headers[0].Height,
		ParentWeight:             headers[0].ParentWeight.String(),
		ParentBaseFee:            headers[0].ParentBaseFee.String(),
		ParentStaterootCid:       headers[0].ParentStateRoot,
		ParentMessageReceiptsCid: headers[0].ParentMessageReceipts,
		ParentTipSetCids:         headers[0].Parents,
		HeaderBlocks:             make([]pgchainbs.DestructuredFilTipSetHeaderBlock, len(headers)),
		BeaconRoundAndData:       make([][]byte, len(headers[0].BeaconEntries)),
	}

	for i, be := range headers[0].BeaconEntries {
		unit := make([]byte, 8, 8+len(be.Data))
		binary.BigEndian.PutUint64(unit, be.Round)
		tsd.BeaconRoundAndData[i] = append(unit, be.Data...)
	}

	for i, hdr := range headers {

		tsd.HeaderBlocks[i].HeaderCid = hdr.Cid()
		tsd.HeaderBlocks[i].MessagesCid = hdr.Messages
		tsd.HeaderBlocks[i].TicketProof = hdr.Ticket.VRFProof

		if hdr.Miner.Protocol() != address.ID {
			return nil, xerrors.Errorf("unexpected miner address '%s' in chain block %s", hdr.Miner, hdr.Cid())
		}

		var viDecodeLen int
		tsd.HeaderBlocks[i].MinerActorID, viDecodeLen = binary.Uvarint(hdr.Miner.Payload())
		if viDecodeLen <= 0 {
			return nil, xerrors.Errorf("failed to parse address payload '%s', binary.Uvarint() returned %d", hdr.Miner, viDecodeLen)
		}

		forkSignal := make([]byte, 10)
		tsd.HeaderBlocks[i].ForkSignalVarint = forkSignal[:binary.PutUvarint(forkSignal, hdr.ForkSignaling)]

		if hdr.BlockSig != nil {
			signatureUnit := make([]byte, 1, 1+len(hdr.BlockSig.Data))
			signatureUnit[0] = byte(hdr.BlockSig.Type)
			tsd.HeaderBlocks[i].TypedSignature = append(signatureUnit, hdr.BlockSig.Data...)
		} else if hdr.Height == 0 {
			// the very very first header has no valid signature: give it something bogus
			tsd.HeaderBlocks[i].TypedSignature = []byte{255}
		} else {
			return nil, xerrors.Errorf("unexpectedly missing signature on chain header %s", hdr.Cid().String())
		}

		electionUnit := make([]byte, 8, 8+len(hdr.ElectionProof.VRFProof))
		binary.BigEndian.PutUint64(electionUnit, uint64(hdr.ElectionProof.WinCount))
		tsd.HeaderBlocks[i].ElectionWincountAndProof = append(electionUnit, hdr.ElectionProof.VRFProof...)

		winpostUnits := make([][]byte, 0, len(hdr.WinPoStProof))
		for _, wpp := range hdr.WinPoStProof {
			unit := make([]byte, 8, 8+len(wpp.ProofBytes))
			binary.BigEndian.PutUint64(unit, uint64(wpp.PoStProof))
			winpostUnits = append(
				winpostUnits,
				append(unit, wpp.ProofBytes...),
			)
		}
		tsd.HeaderBlocks[i].WinpostTypesAndProof = winpostUnits

		// FIXME - perhaps remove this coherence sanity check some day...
		if i != 0 {

			switch {

			case !beaconArrsEqual(hdr.BeaconEntries, headers[0].BeaconEntries):
				return nil, xerrors.Errorf("unexpected Beaconentries mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
					hdr.Cid().String(), i, hdr.BeaconEntries,
					headers[0].Cid().String(), headers[0].BeaconEntries,
				)

			case !cidArrsEqual(hdr.Parents, headers[0].Parents):
				return nil, xerrors.Errorf("unexpected Parents mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
					hdr.Cid().String(), i, hdr.Parents,
					headers[0].Cid().String(), headers[0].Parents,
				)

			case hdr.ParentWeight.String() != headers[0].ParentWeight.String():
				return nil, xerrors.Errorf("unexpected Weight mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
					hdr.Cid().String(), i, hdr.ParentWeight.String(),
					headers[0].Cid().String(), headers[0].ParentWeight.String(),
				)

			case hdr.Height != headers[0].Height:
				return nil, xerrors.Errorf("unexpected Epoch mismatch in tipset: %s(%d)=%d vs %s(0)=%d",
					hdr.Cid().String(), i, hdr.Height,
					headers[0].Cid().String(), headers[0].Height,
				)

			case hdr.ParentStateRoot.String() != headers[0].ParentStateRoot.String():
				return nil, xerrors.Errorf("unexpected ParentStateroot mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
					hdr.Cid().String(), i, hdr.ParentStateRoot.String(),
					headers[0].Cid().String(), headers[0].ParentStateRoot.String(),
				)

			case hdr.ParentMessageReceipts.String() != headers[0].ParentMessageReceipts.String():
				return nil, xerrors.Errorf("unexpected ParentMessageReceipts mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
					hdr.Cid().String(), i, hdr.ParentMessageReceipts.String(),
					headers[0].Cid().String(), headers[0].ParentMessageReceipts.String(),
				)

			case hdr.Timestamp != headers[0].Timestamp:
				return nil, xerrors.Errorf("unexpected Timestamp mismatch in tipset: %s(%d)=%d vs %s(0)=%d",
					hdr.Cid().String(), i, hdr.Timestamp,
					headers[0].Cid().String(), headers[0].Timestamp,
				)

			case hdr.ParentBaseFee.String() != headers[0].ParentBaseFee.String():
				return nil, xerrors.Errorf("unexpected BaseFee mismatch in tipset: %s(%d)=%s vs %s(0)=%s",
					hdr.Cid().String(), i, hdr.ParentBaseFee.String(),
					headers[0].Cid().String(), headers[0].ParentBaseFee.String(),
				)
			}
		}
	}

	return tsd, nil
}

func cidArrsEqual(a, b []cid.Cid) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].KeyString() != b[i].KeyString() {
			return false
		}
	}
	return true
}

func beaconArrsEqual(a, b []types.BeaconEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Round != b[i].Round {
			return false
		}
		if !bytes.Equal(a[i].Data, b[i].Data) {
			return false
		}
	}
	return true
}

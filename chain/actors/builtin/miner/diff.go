package miner

import (
	"context"

	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-amt-ipld/v3"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

func DiffPreCommits(ctx context.Context, store adt.Store, pre, cur State) (*PreCommitChanges, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DiffPreCommits")
	defer span.End()
	prep, err := pre.precommits()
	if err != nil {
		return nil, err
	}

	curp, err := cur.precommits()
	if err != nil {
		return nil, err
	}

	preOpts, err := adt.MapOptsForActorCode(pre.Code())
	if err != nil {
		return nil, err
	}
	curOpts, err := adt.MapOptsForActorCode(cur.Code())
	if err != nil {
		return nil, err
	}

	diffContainer := NewPreCommitDiffContainer(pre, cur)
	if mapRequiresLegacyDiffing(pre, cur, preOpts, curOpts) {
		if span.IsRecording() {
			span.SetAttributes(attribute.String("diff", "slow"))
		}
		err = diff.CompareMap(prep, curp, diffContainer)
		if err != nil {
			return nil, xerrors.Errorf("diff miner precommit: %w", err)
		}
		return diffContainer.Results, nil
	}
	if span.IsRecording() {
		span.SetAttributes(attribute.String("diff", "fast"))
	}

	changes, err := diff.Hamt(ctx, prep, curp, store, store, hamt.UseHashFunction(hamt.HashFunc(preOpts.HashFunc)), hamt.UseTreeBitWidth(preOpts.Bitwidth))
	if err != nil {
		return nil, err
	}
	for _, change := range changes {
		switch change.Type {
		case hamt.Add:
			if err := diffContainer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case hamt.Remove:
			if err := diffContainer.Remove(change.Key, change.Before); err != nil {
				return nil, err
			}
		case hamt.Modify:
			if err := diffContainer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		}
	}

	return diffContainer.Results, nil
}

func NewPreCommitDiffContainer(pre, cur State) *preCommitDiffContainer {
	return &preCommitDiffContainer{
		Results: new(PreCommitChanges),
		pre:     pre,
		after:   cur,
	}
}

type preCommitDiffContainer struct {
	Results    *PreCommitChanges
	pre, after State
}

func (m *preCommitDiffContainer) AsKey(key string) (abi.Keyer, error) {
	sector, err := abi.ParseUIntKey(key)
	if err != nil {
		return nil, xerrors.Errorf("pre commit diff container as key: %w", err)
	}
	return abi.UIntKey(sector), nil
}

func (m *preCommitDiffContainer) Add(key string, val *cbg.Deferred) error {
	sp, err := m.after.decodeSectorPreCommitOnChainInfo(val)
	if err != nil {
		return xerrors.Errorf("pre commit diff container add: %w", err)
	}
	m.Results.Added = append(m.Results.Added, sp)
	return nil
}

func (m *preCommitDiffContainer) Modify(key string, from, to *cbg.Deferred) error {
	return nil
}

func (m *preCommitDiffContainer) Remove(key string, val *cbg.Deferred) error {
	sp, err := m.pre.decodeSectorPreCommitOnChainInfo(val)
	if err != nil {
		return xerrors.Errorf("pre commit diff container remove: %w", err)
	}
	m.Results.Removed = append(m.Results.Removed, sp)
	return nil
}

func DiffSectors(ctx context.Context, store adt.Store, pre, cur State) (*SectorChanges, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DiffSectors")
	defer span.End()
	pres, err := pre.sectors()
	if err != nil {
		return nil, err
	}

	curs, err := cur.sectors()
	if err != nil {
		return nil, err
	}

	preBw := pre.SectorsAmtBitwidth()
	curBw := cur.SectorsAmtBitwidth()
	diffContainer := NewSectorDiffContainer(pre, cur)
	if arrayRequiresLegacyDiffing(pre, cur, preBw, curBw) {
		if span.IsRecording() {
			span.SetAttributes(attribute.String("diff", "slow"))
		}
		err = diff.CompareArray(pres, curs, diffContainer)
		if err != nil {
			return nil, xerrors.Errorf("diff miner sectors: %w", err)
		}
		return diffContainer.Results, nil
	}
	changes, err := diff.Amt(ctx, pres, curs, store, store, amt.UseTreeBitWidth(uint(preBw)))
	if err != nil {
		return nil, err
	}
	if span.IsRecording() {
		span.SetAttributes(attribute.String("diff", "fast"))
	}

	for _, change := range changes {
		switch change.Type {
		case amt.Add:
			if err := diffContainer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case amt.Remove:
			if err := diffContainer.Remove(change.Key, change.Before); err != nil {
				return nil, err
			}
		case amt.Modify:
			if err := diffContainer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		}
	}

	return diffContainer.Results, nil
}

func NewSectorDiffContainer(pre, cur State) *sectorDiffContainer {
	return &sectorDiffContainer{
		Results: new(SectorChanges),
		pre:     pre,
		after:   cur,
	}
}

type sectorDiffContainer struct {
	Results    *SectorChanges
	pre, after State
}

func (m *sectorDiffContainer) Add(key uint64, val *cbg.Deferred) error {
	si, err := m.after.decodeSectorOnChainInfo(val)
	if err != nil {
		return xerrors.Errorf("sector diff container add: %w", err)
	}
	m.Results.Added = append(m.Results.Added, si)
	return nil
}

func (m *sectorDiffContainer) Modify(key uint64, from, to *cbg.Deferred) error {
	siFrom, err := m.pre.decodeSectorOnChainInfo(from)
	if err != nil {
		return xerrors.Errorf("sector diff container modify from: %w", err)
	}

	siTo, err := m.after.decodeSectorOnChainInfo(to)
	if err != nil {
		return xerrors.Errorf("sector diff container modify to: %w", err)
	}

	if siFrom.Expiration != siTo.Expiration {
		m.Results.Extended = append(m.Results.Extended, SectorModification{
			From: siFrom,
			To:   siTo,
		})
	}

	// nullable cid's.....gross
	// if from is null and to isn't that means the sector was Snapped.
	if siFrom.SectorKeyCID == nil && siTo.SectorKeyCID != nil {
		m.Results.Snapped = append(m.Results.Snapped, SectorModification{
			From: siFrom,
			To:   siTo,
		})
	}
	return nil
}

func (m *sectorDiffContainer) Remove(key uint64, val *cbg.Deferred) error {
	si, err := m.pre.decodeSectorOnChainInfo(val)
	if err != nil {
		return xerrors.Errorf("sector diff container remove: %w", err)
	}
	m.Results.Removed = append(m.Results.Removed, si)
	return nil
}

func arrayRequiresLegacyDiffing(pre, cur State, pOpts, cOpts int) bool {
	// amt/v3 cannot read amt/v2 nodes. Their Pointers struct has changed cbor marshalers.
	if pre.Code().Equals(builtin0.StorageMinerActorCodeID) {
		return true
	}
	if pre.Code().Equals(builtin2.StorageMinerActorCodeID) {
		return true
	}
	if cur.Code().Equals(builtin0.StorageMinerActorCodeID) {
		return true
	}
	if cur.Code().Equals(builtin2.StorageMinerActorCodeID) {
		return true
	}
	// bitwidth differences requires legacy diffing.
	if pOpts != cOpts {
		return true
	}
	return false
}

func mapRequiresLegacyDiffing(pre, cur State, pOpts, cOpts *adt.MapOpts) bool {
	// hamt/v3 cannot read hamt/v2 nodes. Their Pointers struct has changed cbor marshalers.
	if pre.Code().Equals(builtin0.StorageMinerActorCodeID) {
		return true
	}
	if pre.Code().Equals(builtin2.StorageMinerActorCodeID) {
		return true
	}
	if cur.Code().Equals(builtin0.StorageMinerActorCodeID) {
		return true
	}
	if cur.Code().Equals(builtin2.StorageMinerActorCodeID) {
		return true
	}
	// bitwidth or hashfunction differences mean legacy diffing.
	if !pOpts.Equal(cOpts) {
		return true
	}
	return false
}

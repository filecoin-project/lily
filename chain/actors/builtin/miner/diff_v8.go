package miner

import (
	"context"
	"fmt"

	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	miner8 "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

func DiffPreCommitsV8(ctx context.Context, store adt.Store, pre, cur State) (*PreCommitChangesV8, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DiffPreCommits")
	defer span.End()
	prep, err := pre.PrecommitsMap()
	if err != nil {
		return nil, err
	}

	curp, err := cur.PrecommitsMap()
	if err != nil {
		return nil, err
	}

	prepR, err := prep.Root()
	if err != nil {
		return nil, err
	}

	curpR, err := curp.Root()
	if err != nil {
		return nil, err
	}

	diffContainer := NewPreCommitDiffContainerV8(pre, cur)
	if prepR.Equals(curpR) {
		return diffContainer.Results, nil
	}

	if MapRequiresLegacyDiffing(pre, cur,
		&adt.MapOpts{
			Bitwidth: pre.SectorsAmtBitwidth(),
			HashFunc: pre.PrecommitsMapHashFunction(),
		},
		&adt.MapOpts{
			Bitwidth: cur.PrecommitsMapBitWidth(),
			HashFunc: cur.PrecommitsMapHashFunction(),
		}) {
		if span.IsRecording() {
			span.SetAttributes(attribute.String("diff", "slow"))
		}
		err = diff.CompareMap(prep, curp, diffContainer)
		if err != nil {
			return nil, fmt.Errorf("diff miner precommit: %w", err)
		}
		return diffContainer.Results, nil
	}
	if span.IsRecording() {
		span.SetAttributes(attribute.String("diff", "fast"))
	}

	changes, err := diff.Hamt(ctx, prep, curp, store, store, hamt.UseHashFunction(pre.PrecommitsMapHashFunction()), hamt.UseTreeBitWidth(pre.PrecommitsMapBitWidth()))
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

type PreCommitChangesV8 struct {
	Added   []miner8.SectorPreCommitOnChainInfo
	Removed []miner8.SectorPreCommitOnChainInfo
}

func MakePreCommitChangesV8() *PreCommitChangesV8 {
	return &PreCommitChangesV8{
		Added:   []miner8.SectorPreCommitOnChainInfo{},
		Removed: []miner8.SectorPreCommitOnChainInfo{},
	}
}

func NewPreCommitDiffContainerV8(pre, cur State) *preCommitDiffContainerV8 {
	return &preCommitDiffContainerV8{
		Results: MakePreCommitChangesV8(),
		pre:     pre,
		after:   cur,
	}
}

type preCommitDiffContainerV8 struct {
	Results    *PreCommitChangesV8
	pre, after State
}

func (m *preCommitDiffContainerV8) AsKey(key string) (abi.Keyer, error) {
	sector, err := abi.ParseUIntKey(key)
	if err != nil {
		return nil, fmt.Errorf("pre commit diff container as key: %w", err)
	}
	return abi.UIntKey(sector), nil
}

func (m *preCommitDiffContainerV8) Add(key string, val *cbg.Deferred) error {
	sp, err := m.after.DecodeSectorPreCommitOnChainInfoToV8(val)
	if err != nil {
		return fmt.Errorf("pre commit diff container add: %w", err)
	}
	m.Results.Added = append(m.Results.Added, sp)
	return nil
}

func (m *preCommitDiffContainerV8) Modify(key string, from, to *cbg.Deferred) error {
	return nil
}

func (m *preCommitDiffContainerV8) Remove(key string, val *cbg.Deferred) error {
	sp, err := m.pre.DecodeSectorPreCommitOnChainInfoToV8(val)
	if err != nil {
		return fmt.Errorf("pre commit diff container remove: %w", err)
	}
	m.Results.Removed = append(m.Results.Removed, sp)
	return nil
}

package datacap

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
)

type BalanceInfo struct {
	Address address.Address
	DataCap abi.StoragePower
}

type BalanceChange struct {
	Before BalanceInfo
	After  BalanceInfo
}

type BalanceChanges struct {
	Added    []BalanceInfo
	Modified []BalanceChange
	Removed  []BalanceInfo
}

func DiffDataCapBalances(ctx context.Context, store adt.Store, pre, cur State) (*BalanceChanges, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DiffDataCapBalances")
	defer span.End()

	preMap, err := pre.VerifiedClients()
	if err != nil {
		return nil, err
	}

	curMap, err := cur.VerifiedClients()
	if err != nil {
		return nil, err
	}

	return diffBalanceMap(ctx, store, pre, cur, preMap, curMap,
		&adt.MapOpts{
			Bitwidth: pre.VerifiedClientsMapBitWidth(),
			HashFunc: pre.VerifiedClientsMapHashFunction(),
		},
		&adt.MapOpts{
			Bitwidth: cur.VerifiedClientsMapBitWidth(),
			HashFunc: cur.VerifiedClientsMapHashFunction(),
		},
	)
}

func diffBalanceMap(ctx context.Context, store adt.Store, pre, cur State, preM, curM adt.Map, preOpts, curOpts *adt.MapOpts) (*BalanceChanges, error) {
	ctx, span := otel.Tracer("").Start(ctx, "diffStates")
	defer span.End()

	diffContainer := NewBalanceDiffContainer(pre, cur)
	if mapRequiresLegacyDiffing(pre, cur, preOpts, curOpts) {
		if span.IsRecording() {
			span.SetAttributes(attribute.String("diff", "slow"))
		}
		if err := diff.CompareMap(preM, curM, diffContainer); err != nil {
			return nil, err
		}
		return diffContainer.Results, nil
	}
	if span.IsRecording() {
		span.SetAttributes(attribute.String("diff", "fast"))
	}

	changes, err := diff.Hamt(ctx, preM, curM, store, store, hamt.UseHashFunction(hamt.HashFunc(preOpts.HashFunc)), hamt.UseTreeBitWidth(preOpts.Bitwidth))
	if err != nil {
		return nil, err
	}
	for _, change := range changes {
		switch change.Type {
		case hamt.Add:
			if err := diffContainer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case hamt.Modify:
			if err := diffContainer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		case hamt.Remove:
			if err := diffContainer.Add(change.Key, change.Before); err != nil {
				return nil, err
			}
		}
	}

	return diffContainer.Results, nil

}

func NewBalanceDiffContainer(pre, cur State) *balanceDiffContainer {
	return &balanceDiffContainer{
		Results: new(BalanceChanges),
		pre:     pre,
		after:   cur,
	}
}

type balanceDiffContainer struct {
	Results    *BalanceChanges
	pre, after State
}

func (m *balanceDiffContainer) AsKey(key string) (abi.Keyer, error) {
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return nil, err
	}
	return abi.AddrKey(addr), nil
}

func (m *balanceDiffContainer) Add(key string, val *cbg.Deferred) error {
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}
	var sp abi.StoragePower
	if err := sp.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return err
	}
	m.Results.Added = append(m.Results.Added, BalanceInfo{
		Address: addr,
		DataCap: sp,
	})
	return nil
}

func (m *balanceDiffContainer) Modify(key string, before, after *cbg.Deferred) error {
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}
	var bsp abi.StoragePower
	if err := bsp.UnmarshalCBOR(bytes.NewReader(before.Raw)); err != nil {
		return err
	}
	var asp abi.StoragePower
	if err := asp.UnmarshalCBOR(bytes.NewReader(after.Raw)); err != nil {
		return err
	}
	m.Results.Modified = append(m.Results.Modified, BalanceChange{
		Before: BalanceInfo{
			Address: addr,
			DataCap: bsp,
		},
		After: BalanceInfo{
			Address: addr,
			DataCap: asp,
		},
	})
	return nil
}

func (m *balanceDiffContainer) Remove(key string, val *cbg.Deferred) error {
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}
	var sp abi.StoragePower
	if err := sp.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return err
	}
	m.Results.Removed = append(m.Results.Removed, BalanceInfo{
		Address: addr,
		DataCap: sp,
	})
	return nil
}

func mapRequiresLegacyDiffing(pre, cur State, pOpts, cOpts *adt.MapOpts) bool {
	// bitwidth or hashfunction differences mean legacy diffing.
	if !pOpts.Equal(cOpts) {
		return true
	}
	return false
}

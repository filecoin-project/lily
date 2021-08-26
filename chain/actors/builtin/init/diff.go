package init

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	logging "github.com/ipfs/go-log/v2"
	typegen "github.com/whyrusleeping/cbor-gen"
)

var log = logging.Logger("visor/actor/init")

type AddressMapChanges struct {
	Added    []AddressPair
	Modified []AddressChange
	Removed  []AddressPair
}

func DiffAddressMap(ctx context.Context, store adt.Store, pre, cur State) (*AddressMapChanges, error) {
	preOpts, err := adt.MapOptsForActorCode(pre.Code())
	if err != nil {
		return nil, err
	}

	curOpts, err := adt.MapOptsForActorCode(cur.Code())
	if err != nil {
		return nil, err
	}

	prem, err := pre.addressMap()
	if err != nil {
		return nil, err
	}

	curm, err := cur.addressMap()
	if err != nil {
		return nil, err
	}

	mapDiffer := NewAddressMapDiffer(pre, cur)
	if requiresLegacyDiffing(pre, cur, preOpts, curOpts) {
		log.Warnw("actor HAMT opts differ, running slower generic map diff", "preCID", pre.Code(), "curCID", cur.Code())
		if err := diff.CompareMap(prem, curm, mapDiffer); err != nil {
			return nil, err
		}
		return mapDiffer.Results, nil
	}

	changes, err := diff.Hamt(ctx, prem, curm, store, store, hamt.UseTreeBitWidth(preOpts.Bitwidth), hamt.UseHashFunction(hamt.HashFunc(preOpts.HashFunc)))
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		switch change.Type {
		case hamt.Add:
			if err := mapDiffer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case hamt.Remove:
			if err := mapDiffer.Remove(change.Key, change.Before); err != nil {
				return nil, err
			}
		case hamt.Modify:
			if err := mapDiffer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		}
	}

	return mapDiffer.Results, nil
}

func requiresLegacyDiffing(pre, cur State, pOpts, cOpts *adt.MapOpts) bool {
	// hamt/v3 cannot read hamt/v2 nodes. Their Pointers struct has changed cbor marshalers.
	if pre.Code() == builtin0.InitActorCodeID {
		return true
	}
	if pre.Code() == builtin2.InitActorCodeID {
		return true
	}
	if cur.Code() == builtin0.InitActorCodeID {
		return true
	}
	if cur.Code() == builtin2.InitActorCodeID {
		return true
	}
	// bitwidth or hashfunction differences mean legacy diffing.
	if !pOpts.Equal(cOpts) {
		return true
	}
	return false
}

func NewAddressMapDiffer(pre, cur State) *addressMapDiffer {
	results := new(AddressMapChanges)
	return &addressMapDiffer{results, pre, cur}
}

type addressMapDiffer struct {
	Results    *AddressMapChanges
	pre, adter State
}

func (i *addressMapDiffer) AsKey(key string) (abi.Keyer, error) {
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return nil, err
	}
	return abi.AddrKey(addr), nil
}

func (i *addressMapDiffer) Add(key string, val *typegen.Deferred) error {
	pkAddr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}
	id := new(typegen.CborInt)
	if err := id.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return err
	}
	idAddr, err := address.NewIDAddress(uint64(*id))
	if err != nil {
		return err
	}
	i.Results.Added = append(i.Results.Added, AddressPair{
		ID: idAddr,
		PK: pkAddr,
	})
	return nil
}

func (i *addressMapDiffer) Modify(key string, before, after *typegen.Deferred) error {
	pkAddr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}

	fromID := new(typegen.CborInt)
	if err := fromID.UnmarshalCBOR(bytes.NewReader(before.Raw)); err != nil {
		return err
	}
	fromIDAddr, err := address.NewIDAddress(uint64(*fromID))
	if err != nil {
		return err
	}

	toID := new(typegen.CborInt)
	if err := toID.UnmarshalCBOR(bytes.NewReader(after.Raw)); err != nil {
		return err
	}
	toIDAddr, err := address.NewIDAddress(uint64(*toID))
	if err != nil {
		return err
	}

	i.Results.Modified = append(i.Results.Modified, AddressChange{
		From: AddressPair{
			ID: fromIDAddr,
			PK: pkAddr,
		},
		To: AddressPair{
			ID: toIDAddr,
			PK: pkAddr,
		},
	})
	return nil
}

func (i *addressMapDiffer) Remove(key string, val *typegen.Deferred) error {
	pkAddr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}
	id := new(typegen.CborInt)
	if err := id.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return err
	}
	idAddr, err := address.NewIDAddress(uint64(*id))
	if err != nil {
		return err
	}
	i.Results.Removed = append(i.Results.Removed, AddressPair{
		ID: idAddr,
		PK: pkAddr,
	})
	return nil
}

type AddressChange struct {
	From AddressPair
	To   AddressPair
}

type AddressPair struct {
	ID address.Address
	PK address.Address
}
